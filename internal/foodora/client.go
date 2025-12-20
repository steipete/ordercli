package foodora

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL *url.URL
	http    *http.Client

	deviceID       string
	globalEntityID string
	targetISO      string
	cookieHeader   string
	fpAPIKey       string
	appName        string
	originalUA     string

	accessToken string
	userAgent   string
}

type Options struct {
	BaseURL           string
	DeviceID          string
	GlobalEntityID    string
	TargetCountryISO  string
	AccessToken       string
	UserAgent         string
	CookieHeader      string
	FPAPIKey          string
	AppName           string
	OriginalUserAgent string
}

func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, errors.New("base URL not set (run `ordercli foodora config set ...`)")
	}
	u, err := url.Parse(opts.BaseURL)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	ua := opts.UserAgent
	if ua == "" {
		ua = "ordercli"
	}

	return &Client{
		baseURL: u,
		http: &http.Client{
			Timeout: 20 * time.Second,
		},
		deviceID:       opts.DeviceID,
		globalEntityID: opts.GlobalEntityID,
		targetISO:      opts.TargetCountryISO,
		cookieHeader:   opts.CookieHeader,
		fpAPIKey:       opts.FPAPIKey,
		appName:        opts.AppName,
		originalUA:     opts.OriginalUserAgent,
		accessToken:    opts.AccessToken,
		userAgent:      ua,
	}, nil
}

func (c *Client) SetAccessToken(token string) { c.accessToken = token }

func (c *Client) OAuthTokenPassword(ctx context.Context, req OAuthPasswordRequest) (AuthToken, *MfaChallenge, error) {
	values := url.Values{}
	values.Set("username", req.Username)
	values.Set("password", req.Password)
	values.Set("grant_type", "password")
	values.Set("client_secret", req.ClientSecret)
	values.Set("scope", "API_CUSTOMER")
	clientID := req.ClientID
	if clientID == "" {
		clientID = "android"
	}
	values.Set("client_id", clientID)

	return c.oauthToken(ctx, values, oauthHeaders{
		otpMethod: req.OTPMethod,
		otpCode:   req.OTPCode,
		mfaToken:  req.MfaToken,
	})
}

func (c *Client) OAuthTokenRefresh(ctx context.Context, req OAuthRefreshRequest) (AuthToken, error) {
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", req.RefreshToken)
	values.Set("client_secret", req.ClientSecret)
	values.Set("scope", "API_CUSTOMER")
	clientID := req.ClientID
	if clientID == "" {
		clientID = "android"
	}
	values.Set("client_id", clientID)

	token, _, err := c.oauthToken(ctx, values, oauthHeaders{
		otpMethod: "",
	})
	return token, err
}

func (c *Client) ActiveOrders(ctx context.Context) (ActiveOrdersResponse, error) {
	var out ActiveOrdersResponse
	if err := c.getJSON(ctx, "tracking/active-orders", nil, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) OrderStatus(ctx context.Context, orderCode string) (OrderStatusResponse, error) {
	var out OrderStatusResponse
	q := url.Values{}
	q.Set("q", "0")
	q.Set("show_map_early_variation", "Variation1")
	q.Set("vendor_details_variation", "Variation1")
	path := fmt.Sprintf("tracking/orders/%s", url.PathEscape(orderCode))
	if err := c.getJSON(ctx, path, q, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) OrderHistory(ctx context.Context, req OrderHistoryRequest) (OrderHistoryResponse, error) {
	var out OrderHistoryResponse
	q := url.Values{}
	include := req.Include
	if include == "" {
		include = "order_products,order_details"
	}
	q.Set("include", include)
	q.Set("offset", strconv.Itoa(max(0, req.Offset)))
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("pandago_enabled", strconv.FormatBool(req.PandaGoEnabled))
	if err := c.getJSON(ctx, "orders/order_history", q, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) OrderHistoryByCode(ctx context.Context, req OrderHistoryByCodeRequest) (OrderHistoryRawResponse, error) {
	var out OrderHistoryRawResponse
	if strings.TrimSpace(req.OrderCode) == "" {
		return out, errors.New("order history: missing order code")
	}
	q := url.Values{}
	include := req.Include
	if include == "" {
		include = "order_products,order_details"
	}
	q.Set("include", include)
	q.Set("order_code", req.OrderCode)
	q.Set("item_replacement", strconv.FormatBool(req.ItemReplacement))

	if err := c.getJSON(ctx, "orders/order_history", q, &out); err != nil {
		return out, err
	}
	return out, nil
}

type oauthHeaders struct {
	otpMethod string
	otpCode   string
	mfaToken  string
}

func (c *Client) oauthToken(ctx context.Context, form url.Values, h oauthHeaders) (AuthToken, *MfaChallenge, error) {
	u := c.baseURL.ResolveReference(&url.URL{Path: "oauth2/token"})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return AuthToken{}, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if c.originalUA != "" {
		req.Header.Set("X-Original-User-Agent", c.originalUA)
	}
	if c.deviceID != "" {
		req.Header.Set("X-Device", c.deviceID)
		req.Header.Set("Device-Id", c.deviceID)
	}
	if c.cookieHeader != "" {
		req.Header.Set("Cookie", c.cookieHeader)
	}
	if c.fpAPIKey != "" {
		req.Header.Set("X-FP-API-KEY", c.fpAPIKey)
	}
	if c.appName != "" {
		req.Header.Set("App-Name", c.appName)
	}
	req.Header.Set("X-OTP-Method", h.otpMethod)
	if h.otpCode != "" {
		req.Header.Set("X-OTP", h.otpCode)
	}
	if h.mfaToken != "" {
		req.Header.Set("X-Mfa-Token", h.mfaToken)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return AuthToken{}, nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return AuthToken{}, nil, err
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		var token AuthToken
		if err := json.Unmarshal(body, &token); err != nil {
			return AuthToken{}, nil, fmt.Errorf("oauth2/token: decode success body: %w", err)
		}
		if token.AccessToken == "" {
			return AuthToken{}, nil, fmt.Errorf("oauth2/token: missing access_token (status %d)", res.StatusCode)
		}
		return token, nil, nil
	}

	ch, ok := parseMfaTriggered(body, res.Header)
	if ok {
		return AuthToken{}, &ch, nil
	}

	return AuthToken{}, nil, &HTTPError{
		Method:     req.Method,
		URL:        req.URL.String(),
		StatusCode: res.StatusCode,
		Body:       body,
	}
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL.ResolveReference(&url.URL{Path: path})
	if query != nil && len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if c.originalUA != "" {
		req.Header.Set("X-Original-User-Agent", c.originalUA)
	}
	if c.deviceID != "" {
		req.Header.Set("X-Device", c.deviceID)
		req.Header.Set("Device-Id", c.deviceID)
	}
	if c.cookieHeader != "" {
		req.Header.Set("Cookie", c.cookieHeader)
	}
	if c.fpAPIKey != "" {
		req.Header.Set("X-FP-API-KEY", c.fpAPIKey)
	}
	if c.appName != "" {
		req.Header.Set("App-Name", c.appName)
	}
	if c.globalEntityID != "" {
		req.Header.Set("X-Global-Entity-ID", c.globalEntityID)
	}
	if c.targetISO != "" {
		req.Header.Set("X-Target-Country-Code-ISO", c.targetISO)
	}
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return &HTTPError{
			Method:     req.Method,
			URL:        req.URL.String(),
			StatusCode: res.StatusCode,
			Body:       body,
		}
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err == nil {
		return nil
	}

	// Fallback: tolerate API drift by decoding as lenient JSON.
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s: decode JSON: %w", path, err)
	}
	return nil
}

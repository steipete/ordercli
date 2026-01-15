package glovo

import (
	"context"
	"crypto/rand"
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

// Client is a Glovo API client
type Client struct {
	baseURL      *url.URL
	http         *http.Client
	accessToken  string
	deviceURN    string
	cityCode     string
	countryCode  string
	languageCode string
	sessionID    string
	latitude     float64
	longitude    float64
}

// Options configures a new Glovo client
type Options struct {
	BaseURL     string
	AccessToken string
	DeviceURN   string
	CityCode    string
	CountryCode string
	Language    string
	Latitude    float64
	Longitude   float64
}

// New creates a new Glovo API client
func New(opts Options) (*Client, error) {
	if opts.AccessToken == "" {
		return nil, errors.New("access token not set (run `ordercli glovo session <token>`)")
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "https://api.glovoapp.com"
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	deviceURN := opts.DeviceURN
	if deviceURN == "" {
		deviceURN = "glv:device:" + newUUID()
	}

	sessionID := newUUID()

	lang := opts.Language
	if lang == "" {
		lang = "en"
	}

	return &Client{
		baseURL:      u,
		http:         &http.Client{Timeout: 20 * time.Second},
		accessToken:  opts.AccessToken,
		deviceURN:    deviceURN,
		cityCode:     opts.CityCode,
		countryCode:  opts.CountryCode,
		languageCode: lang,
		sessionID:    sessionID,
		latitude:     opts.Latitude,
		longitude:    opts.Longitude,
	}, nil
}

// setHeaders sets all required Glovo API headers
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	// Required glovo-* headers
	req.Header.Set("glovo-api-version", "14")
	req.Header.Set("glovo-app-context", "web")
	req.Header.Set("glovo-app-development-state", "prod")
	req.Header.Set("glovo-app-platform", "web")
	req.Header.Set("glovo-app-type", "customer")
	req.Header.Set("glovo-app-version", "v1.1782.0")
	req.Header.Set("glovo-client-info", "web-customer-web-react/v1.1782.0 project:customer-web")

	// Location headers
	if c.latitude != 0 {
		req.Header.Set("glovo-delivery-location-latitude", strconv.FormatFloat(c.latitude, 'f', -1, 64))
	}
	if c.longitude != 0 {
		req.Header.Set("glovo-delivery-location-longitude", strconv.FormatFloat(c.longitude, 'f', -1, 64))
	}
	req.Header.Set("glovo-delivery-location-accuracy", "0")
	req.Header.Set("glovo-delivery-location-timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))

	// Device and session
	req.Header.Set("glovo-device-urn", c.deviceURN)
	req.Header.Set("glovo-dynamic-session-id", c.sessionID)

	// Language and location
	req.Header.Set("glovo-language-code", c.languageCode)
	if c.cityCode != "" {
		req.Header.Set("glovo-location-city-code", c.cityCode)
	}
	if c.countryCode != "" {
		req.Header.Set("glovo-location-country-code", c.countryCode)
	}

	// Perseus (tracking) headers
	perseusClientID := newUUID()
	req.Header.Set("glovo-perseus-client-id", perseusClientID)
	req.Header.Set("glovo-perseus-consent", "essential")
	req.Header.Set("glovo-perseus-session-id", c.sessionID)
	req.Header.Set("glovo-perseus-session-timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))

	// Request tracking
	req.Header.Set("glovo-request-id", newUUID())
	req.Header.Set("glovo-request-ttl", "7500")
}

// newUUID generates a UUIDv4-ish string
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("gen-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	)
}

// getJSON performs a GET request and decodes the JSON response.
func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL.ResolveReference(&url.URL{Path: path})
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			Method:     req.Method,
			URL:        req.URL.String(),
			StatusCode: resp.StatusCode,
			Body:       body,
		}
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s: decode JSON: %w", path, err)
	}
	return nil
}

// OrderHistory fetches the order history list
func (c *Client) OrderHistory(ctx context.Context, offset, limit int) (OrdersResponse, error) {
	if limit <= 0 {
		limit = 12
	}

	query := url.Values{
		"offset": {strconv.Itoa(offset)},
		"limit":  {strconv.Itoa(limit)},
	}

	var out OrdersResponse
	if err := c.getJSON(ctx, "v3/customer/orders-list", query, &out); err != nil {
		return OrdersResponse{}, err
	}
	return out, nil
}

// ActiveOrders returns orders that are currently active (being delivered)
func (c *Client) ActiveOrders(ctx context.Context) ([]Order, error) {
	resp, err := c.OrderHistory(ctx, 0, 20)
	if err != nil {
		return nil, err
	}

	var active []Order
	for _, o := range resp.Orders {
		// Active orders have layoutType "ACTIVE_ORDER" or similar non-inactive types
		if o.LayoutType != "INACTIVE_ORDER" {
			active = append(active, o)
		}
	}
	return active, nil
}

// Baskets fetches the user's shopping carts
func (c *Client) Baskets(ctx context.Context, customerID int) (BasketsResponse, error) {
	path := fmt.Sprintf("v1/authenticated/customers/%d/baskets", customerID)

	var out BasketsResponse
	if err := c.getJSON(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Me fetches the current user's profile
func (c *Client) Me(ctx context.Context) (UserResponse, error) {
	var out UserResponse
	if err := c.getJSON(ctx, "v3/me", nil, &out); err != nil {
		return UserResponse{}, err
	}
	return out, nil
}

// GetOrder fetches a single order by ID from history
func (c *Client) GetOrder(ctx context.Context, orderID int) (Order, error) {
	// Glovo doesn't have a direct single-order endpoint, so we search history
	resp, err := c.OrderHistory(ctx, 0, 50)
	if err != nil {
		return Order{}, err
	}

	for _, o := range resp.Orders {
		if o.OrderID == orderID {
			return o, nil
		}
	}

	return Order{}, fmt.Errorf("order %d not found in recent history", orderID)
}

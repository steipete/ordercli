package deliveroo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	http        *http.Client
	market      string
	consumerURL string
	bearerToken string
	cookie      string
}

type ClientOptions struct {
	BaseURL     string
	Market      string
	BearerToken string
	Cookie      string
	Timeout     time.Duration
}

func NewClient(opts ClientOptions) (*Client, error) {
	if strings.TrimSpace(opts.BearerToken) == "" {
		return nil, errors.New("missing bearer token")
	}
	if opts.Timeout == 0 {
		opts.Timeout = 20 * time.Second
	}
	base := strings.TrimSpace(opts.BaseURL)
	if base == "" {
		var err error
		base, err = BuildAPIBaseURL("", strings.TrimSpace(opts.Market))
		if err != nil {
			return nil, err
		}
	}

	consumer, err := ConsumerBaseURL(base)
	if err != nil {
		return nil, err
	}

	return &Client{
		http: &http.Client{
			Timeout: opts.Timeout,
		},
		market:      strings.TrimSpace(opts.Market),
		consumerURL: consumer,
		bearerToken: strings.TrimSpace(opts.BearerToken),
		cookie:      strings.TrimSpace(opts.Cookie),
	}, nil
}

func (c *Client) OrderHistory(ctx context.Context, p OrderHistoryParams) (OrdersResponse, error) {
	u, err := OrderHistoryURL(c.consumerURL, p)
	if err != nil {
		return OrdersResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return OrdersResponse{}, err
	}
	req.Header.Set("Accept", "application/json")

	auth := c.bearerToken
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		auth = "Bearer " + auth
	}
	req.Header.Set("Authorization", auth)
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	if c.market != "" {
		req.Header.Set("X-Deliveroo-Market", c.market)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return OrdersResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return OrdersResponse{}, fmt.Errorf("deliveroo: unauthorized (%d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return OrdersResponse{}, fmt.Errorf("deliveroo: unexpected status %d", resp.StatusCode)
	}

	var out OrdersResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return OrdersResponse{}, err
	}
	return out, nil
}

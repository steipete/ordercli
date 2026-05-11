package glovo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestNewDeviceURNFormat(t *testing.T) {
	got := NewDeviceURN()
	if !regexp.MustCompile(`^glv:device:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(got) {
		t.Fatalf("unexpected device URN: %q", got)
	}
}

func TestClientSetsHeadersAndQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/customer/orders-list" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.URL.Query().Get("offset") != "4" || r.URL.Query().Get("limit") != "7" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		assertHeader := func(name, want string) {
			t.Helper()
			if got := r.Header.Get(name); got != want {
				t.Fatalf("%s=%q want %q", name, got, want)
			}
		}
		assertHeader("Authorization", "Bearer tok")
		assertHeader("glovo-device-urn", "glv:device:fixed")
		assertHeader("glovo-location-city-code", "MAD")
		assertHeader("glovo-location-country-code", "ES")
		assertHeader("glovo-language-code", "es")
		assertHeader("glovo-delivery-location-latitude", "40.4168")
		assertHeader("glovo-delivery-location-longitude", "-3.7038")
		if got := r.Header.Get("glovo-request-id"); got == "" {
			t.Fatal("missing request id")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"pagination":{"currentLimit":7},"orders":[]}`))
	}))
	defer srv.Close()

	c, err := New(Options{
		BaseURL:     srv.URL,
		AccessToken: "tok",
		DeviceURN:   "glv:device:fixed",
		CityCode:    "MAD",
		CountryCode: "ES",
		Language:    "es",
		Latitude:    40.4168,
		Longitude:   -3.7038,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp, err := c.OrderHistory(context.Background(), 4, 7)
	if err != nil {
		t.Fatalf("OrderHistory: %v", err)
	}
	if resp.Pagination.CurrentLimit != 7 {
		t.Fatalf("current limit=%d", resp.Pagination.CurrentLimit)
	}
}

func TestClientHTTPErrorIncludesStatusAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, strings.Repeat("denied", 80), http.StatusUnauthorized)
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Me(context.Background())
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("err=%T %[1]v", err)
	}
	if !httpErr.IsUnauthorized() || httpErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected HTTP error: %#v", httpErr)
	}
	if got := httpErr.Error(); !strings.Contains(got, "HTTP 401") || !strings.Contains(got, "denied") || len(got) > 380 {
		t.Fatalf("unexpected error string: %q", got)
	}
}

func TestClientActiveOrdersFiltersInactiveOrders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/customer/orders-list" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"orders": [
				{"orderId": 1, "layoutType": "INACTIVE_ORDER"},
				{"orderId": 2, "layoutType": "ACTIVE_ORDER"},
				{"orderId": 3, "layoutType": "PENDING_ORDER"}
			]
		}`))
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	active, err := c.ActiveOrders(context.Background())
	if err != nil {
		t.Fatalf("ActiveOrders: %v", err)
	}
	if len(active) != 2 || active[0].OrderID != 2 || active[1].OrderID != 3 {
		t.Fatalf("active=%#v", active)
	}
}

func TestClientBasketsUsesCustomerPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/authenticated/customers/42/baskets" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{
			"storeId": 7,
			"storeName": "Corner Shop",
			"products": [{"id": 1, "name": "Milk", "quantity": 2}],
			"total": 9.5,
			"currency": "EUR"
		}]`))
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	baskets, err := c.Baskets(context.Background(), 42)
	if err != nil {
		t.Fatalf("Baskets: %v", err)
	}
	if len(baskets) != 1 || baskets[0].StoreID != 7 || baskets[0].Products[0].Name != "Milk" {
		t.Fatalf("baskets=%#v", baskets)
	}
}

func TestClientGetOrderFindsRecentOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "50" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"orders": [
				{"orderId": 10, "content": {"title": "Old"}},
				{"orderId": 20, "content": {"title": "Wanted"}}
			]
		}`))
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	order, err := c.GetOrder(context.Background(), 20)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if order.Content.Title != "Wanted" {
		t.Fatalf("order=%#v", order)
	}
}

func TestClientGetOrderMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"orders":[{"orderId":10}]}`))
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.GetOrder(context.Background(), 99)
	if err == nil || !strings.Contains(err.Error(), "order 99 not found") {
		t.Fatalf("err=%v", err)
	}
}

func TestFooterItemDataString(t *testing.T) {
	if got := (&FooterItem{Data: "12,34 EUR"}).DataString(); got != "12,34 EUR" {
		t.Fatalf("string data=%q", got)
	}
	if got := (&FooterItem{Data: map[string]any{"text": "not a string"}}).DataString(); got != "" {
		t.Fatalf("object data=%q", got)
	}
}

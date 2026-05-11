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

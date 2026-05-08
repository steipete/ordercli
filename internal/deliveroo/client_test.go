package deliveroo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_OrderHistory(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Fatalf("missing authorization")
		}
		if !strings.HasPrefix(strings.ToLower(r.Header.Get("Authorization")), "bearer ") {
			t.Fatalf("expected bearer auth, got %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/consumer/order-history/v1/orders" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"orders":[{"id":"o1","order_number":7756,"status":"delivered"}]}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientOptions{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Market:      "at",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := c.OrderHistory(context.Background(), OrderHistoryParams{Offset: 0, Limit: 1})
	if err != nil {
		t.Fatalf("OrderHistory: %v", err)
	}
	if len(resp.Orders) != 1 || resp.Orders[0].ID != "o1" || resp.Orders[0].OrderNumber != "7756" {
		t.Fatalf("unexpected resp: %#v", resp)
	}
}

func TestClient_OrderHistory_Unauthorized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientOptions{BaseURL: srv.URL, BearerToken: "tok"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.OrderHistory(context.Background(), OrderHistoryParams{Offset: 0, Limit: 1})
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("unexpected err: %v", err)
	}
}

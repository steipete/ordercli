package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steipete/ordercli/internal/deliveroo"
)

func TestDeliverooCLI_ConfigAndHistory(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/consumer/order-history/v1/orders" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"orders":[{"id":"o1","status":"delivered","restaurant":{"name":"R"}}]}`))
	}))
	defer srv.Close()

	out, _, err := runCLI(cfgPath, []string{"deliveroo", "config", "set", "--market", "uk", "--base-url", srv.URL}, "")
	if err != nil {
		t.Fatalf("config set: %v out=%s", err, out)
	}
	out, _, err = runCLI(cfgPath, []string{"deliveroo", "config", "show"}, "")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if !strings.Contains(out, "market=uk") || !strings.Contains(out, "base_url="+srv.URL) {
		t.Fatalf("unexpected out: %s", out)
	}

	setEnv(t, "DELIVEROO_BEARER_TOKEN", "tok")

	out, _, err = runCLI(cfgPath, []string{"deliveroo", "history", "--limit", "1"}, "")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if !strings.Contains(out, "id=o1") {
		t.Fatalf("unexpected out: %s", out)
	}

	_, _, err = runCLI(cfgPath, []string{"deliveroo", "orders", "--once"}, "")
	if err != nil {
		t.Fatalf("orders: %v", err)
	}
}

func TestDeliverooCLI_MissingBearerToken(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	_, _, err := runCLI(cfgPath, []string{"deliveroo", "history"}, "")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeliverooCLI_OrdersFallbackToPublicStatus(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	origResolve := deliverooResolveLatestStatusURL
	origFetch := deliverooFetchPublicStatus
	t.Cleanup(func() {
		deliverooResolveLatestStatusURL = origResolve
		deliverooFetchPublicStatus = origFetch
	})

	deliverooResolveLatestStatusURL = func(_ context.Context, browser string) (string, error) {
		if browser != "atlas" {
			t.Fatalf("browser=%q", browser)
		}
		return "https://deliveroo.example/orders/123/status", nil
	}
	deliverooFetchPublicStatus = func(_ context.Context, targetURL string, timeout time.Duration) (deliveroo.PublicStatus, error) {
		if targetURL != "https://deliveroo.example/orders/123/status" {
			t.Fatalf("url=%q", targetURL)
		}
		if timeout <= 0 {
			t.Fatalf("timeout=%s", timeout)
		}
		return deliveroo.PublicStatus{
			URL:              targetURL,
			Restaurant:       "Ta'Mini",
			OrderNumber:      "Order #5040",
			Customer:         "Gildas",
			Status:           "The order is out for delivery",
			EstimatedArrival: "About 6 minutes",
			Courier:          "Rubel",
			Items:            []string{"1x Falafel Wrap"},
		}, nil
	}

	out, _, err := runCLI(cfgPath, []string{"deliveroo", "orders", "--once", "--browser", "atlas"}, "")
	if err != nil {
		t.Fatalf("orders fallback: %v", err)
	}
	for _, want := range []string{
		"restaurant=Ta'Mini",
		"order=Order #5040",
		"for=Gildas",
		"status=The order is out for delivery",
		"eta=About 6 minutes",
		"courier=Rubel",
		"1x Falafel Wrap",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

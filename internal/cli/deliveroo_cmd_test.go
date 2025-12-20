package cli

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
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

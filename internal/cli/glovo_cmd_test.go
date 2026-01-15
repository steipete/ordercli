package cli

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlovoCLI_ConfigSetAndShow(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	// Test config set
	out, _, err := runCLI(cfgPath, []string{"glovo", "config", "set",
		"--city-code", "MAD",
		"--country-code", "ES",
		"--language", "en",
	}, "")
	if err != nil {
		t.Fatalf("config set: %v out=%s", err, out)
	}

	// Test config show
	out, _, err = runCLI(cfgPath, []string{"glovo", "config", "show"}, "")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if !strings.Contains(out, "city_code=MAD") {
		t.Fatalf("missing city_code in output: %s", out)
	}
	if !strings.Contains(out, "country_code=ES") {
		t.Fatalf("missing country_code in output: %s", out)
	}
	if !strings.Contains(out, "language=en") {
		t.Fatalf("missing language in output: %s", out)
	}
}

func TestGlovoCLI_SessionCommand(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	// Set access token
	out, _, err := runCLI(cfgPath, []string{"glovo", "session", "test-token-12345"}, "")
	if err != nil {
		t.Fatalf("session: %v out=%s", err, out)
	}
	if !strings.Contains(out, "access token saved") {
		t.Fatalf("unexpected output: %s", out)
	}

	// Verify token is visible in config show (truncated with ...)
	out, _, err = runCLI(cfgPath, []string{"glovo", "config", "show"}, "")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if !strings.Contains(out, "access_token=test-token-12345...") {
		t.Fatalf("token not visible in config: %s", out)
	}
}

func TestGlovoCLI_History(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check path
		if !strings.HasPrefix(r.URL.Path, "/v3/customer/orders-list") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Return mock order response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"pagination": {"currentLimit": 12, "next": null},
			"orders": [{
				"orderId": 123,
				"orderUrn": "glv:order:test",
				"content": {"title": "Test Restaurant", "body": [{"type": "TEXT", "data": "1 x Item"}]},
				"footer": {"left": {"type": "TEXT", "data": "10,00 EUR"}, "right": null},
				"style": "DEFAULT",
				"layoutType": "INACTIVE_ORDER",
				"image": {"lightImageId": "", "darkImageId": ""}
			}]
		}`))
	}))
	defer srv.Close()

	// Set config with test server URL and token
	_, _, err := runCLI(cfgPath, []string{"glovo", "config", "set", "--base-url", srv.URL}, "")
	if err != nil {
		t.Fatalf("config set: %v", err)
	}
	_, _, err = runCLI(cfgPath, []string{"glovo", "session", "test-token"}, "")
	if err != nil {
		t.Fatalf("session: %v", err)
	}

	// Test history command
	out, _, err := runCLI(cfgPath, []string{"glovo", "history"}, "")
	if err != nil {
		t.Fatalf("history: %v out=%s", err, out)
	}
	if !strings.Contains(out, "Test Restaurant") {
		t.Fatalf("restaurant not found in output: %s", out)
	}
	if !strings.Contains(out, "10,00 EUR") {
		t.Fatalf("price not found in output: %s", out)
	}
}

func TestGlovoCLI_Me(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/me" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": 12345,
			"type": "Customer",
			"urn": "glv:customer:test",
			"name": "Test User",
			"email": "test@example.com",
			"preferredCityCode": "MAD",
			"preferredLanguage": "en",
			"deliveredOrdersCount": 5
		}`))
	}))
	defer srv.Close()

	// Set config
	_, _, err := runCLI(cfgPath, []string{"glovo", "config", "set", "--base-url", srv.URL}, "")
	if err != nil {
		t.Fatalf("config set: %v", err)
	}
	_, _, err = runCLI(cfgPath, []string{"glovo", "session", "test-token"}, "")
	if err != nil {
		t.Fatalf("session: %v", err)
	}

	// Test me command
	out, _, err := runCLI(cfgPath, []string{"glovo", "me"}, "")
	if err != nil {
		t.Fatalf("me: %v out=%s", err, out)
	}
	if !strings.Contains(out, "Test User") {
		t.Fatalf("name not found in output: %s", out)
	}
	if !strings.Contains(out, "test@example.com") {
		t.Fatalf("email not found in output: %s", out)
	}
}

func TestGlovoCLI_MissingToken(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	// Try to run history without token
	_, _, err := runCLI(cfgPath, []string{"glovo", "history"}, "")
	if err == nil {
		t.Fatalf("expected error when token missing")
	}
}

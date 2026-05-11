package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steipete/ordercli/internal/config"
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

	// Verify token is redacted in config show.
	out, _, err = runCLI(cfgPath, []string{"glovo", "config", "show"}, "")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if !strings.Contains(out, "access_token=***") {
		t.Fatalf("token not redacted in config: %s", out)
	}
	if strings.Contains(out, "test-token-12345") {
		t.Fatalf("token leaked in config: %s", out)
	}
}

func TestGlovoCLI_History(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	var deviceURNs []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check path
		if !strings.HasPrefix(r.URL.Path, "/v3/customer/orders-list") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("authorization=%q", got)
		}
		deviceURN := r.Header.Get("glovo-device-urn")
		if !strings.HasPrefix(deviceURN, "glv:device:") {
			t.Errorf("glovo-device-urn=%q", deviceURN)
		}
		deviceURNs = append(deviceURNs, deviceURN)
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

	// Generated device URNs should be persisted and reused across invocations.
	out, _, err = runCLI(cfgPath, []string{"glovo", "history"}, "")
	if err != nil {
		t.Fatalf("history second run: %v out=%s", err, out)
	}
	if len(deviceURNs) != 2 || deviceURNs[0] == "" || deviceURNs[0] != deviceURNs[1] {
		t.Fatalf("device URN not reused: %#v", deviceURNs)
	}
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg config.Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.Providers.Glovo == nil || cfg.Providers.Glovo.DeviceURN != deviceURNs[0] {
		t.Fatalf("device URN not persisted: config=%#v headers=%#v", cfg.Providers.Glovo, deviceURNs)
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

func TestGlovoCLI_ConfigSetRequiresAChange(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	_, _, err := runCLI(cfgPath, []string{"glovo", "config", "set"}, "")
	if err == nil || !strings.Contains(err.Error(), "nothing to set") {
		t.Fatalf("err=%v", err)
	}
}

func TestGlovoCLI_OrderOrdersCartAndLogout(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	courier := "Rider One"

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/customer/orders-list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"pagination": {"currentLimit": 50},
			"orders": [{
				"orderId": 777,
				"orderUrn": "glv:order:777",
				"content": {"title": "Pizza Place", "body": [{"type": "TEXT", "data": "1 x Pizza\n1 x Soda"}]},
				"footer": {"left": {"type": "TEXT", "data": "18,50 EUR"}, "right": null},
				"layoutType": "ACTIVE_ORDER",
				"courierName": "` + courier + `",
				"image": {"lightImageId": "", "darkImageId": ""}
			}]
		}`))
	})
	mux.HandleFunc("/v3/me", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": 42, "name": "Test User", "email": "test@example.com"}`))
	})
	mux.HandleFunc("/v1/authenticated/customers/42/baskets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{
			"storeId": 9,
			"storeName": "Corner Shop",
			"products": [{"id": 1, "name": "Water", "quantity": 3, "totalPrice": 4.5}],
			"subTotal": 4.5,
			"deliveryFee": 1.5,
			"serviceFee": 0.5,
			"total": 6.5,
			"currency": "EUR",
			"minOrderValue": 10,
			"isMinOrderMet": false
		}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if out, _, err := runCLI(cfgPath, []string{"glovo", "config", "set", "--base-url", srv.URL}, ""); err != nil {
		t.Fatalf("config set: %v out=%s", err, out)
	}
	if out, _, err := runCLI(cfgPath, []string{"glovo", "session", "test-token"}, ""); err != nil {
		t.Fatalf("session: %v out=%s", err, out)
	}

	out, _, err := runCLI(cfgPath, []string{"glovo", "order", "777"}, "")
	if err != nil {
		t.Fatalf("order: %v out=%s", err, out)
	}
	for _, want := range []string{"Order ID: 777", "Pizza Place", "18,50 EUR", courier, "1 x Pizza"} {
		if !strings.Contains(out, want) {
			t.Fatalf("order output missing %q: %s", want, out)
		}
	}

	out, _, err = runCLI(cfgPath, []string{"glovo", "orders"}, "")
	if err != nil {
		t.Fatalf("orders: %v out=%s", err, out)
	}
	if !strings.Contains(out, "[777] Pizza Place") || !strings.Contains(out, "Courier: "+courier) {
		t.Fatalf("orders output: %s", out)
	}

	out, _, err = runCLI(cfgPath, []string{"glovo", "cart"}, "")
	if err != nil {
		t.Fatalf("cart: %v out=%s", err, out)
	}
	for _, want := range []string{"Corner Shop", "3x Water", "Delivery: 1.50 EUR", "! Min order: 10.00 EUR"} {
		if !strings.Contains(out, want) {
			t.Fatalf("cart output missing %q: %s", want, out)
		}
	}

	out, _, err = runCLI(cfgPath, []string{"glovo", "logout"}, "")
	if err != nil {
		t.Fatalf("logout: %v out=%s", err, out)
	}
	if !strings.Contains(out, "logged out") {
		t.Fatalf("logout output: %s", out)
	}
	out, _, err = runCLI(cfgPath, []string{"glovo", "config", "show"}, "")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if !strings.Contains(out, "access_token=(not set)") || strings.Contains(out, "device_urn=***") {
		t.Fatalf("logout did not clear auth/device config: %s", out)
	}
}

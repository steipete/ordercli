package browserauth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steipete/ordercli/internal/foodora"
)

func TestOAuthTokenPassword_Validation(t *testing.T) {
	_, _, _, err := OAuthTokenPassword(context.Background(), foodora.OAuthPasswordRequest{}, PasswordOptions{})
	if err == nil || !strings.Contains(err.Error(), "base URL missing") {
		t.Fatalf("expected base url missing, got %v", err)
	}

	_, _, _, err = OAuthTokenPassword(context.Background(), foodora.OAuthPasswordRequest{}, PasswordOptions{BaseURL: "https://x.example/"})
	if err == nil || !strings.Contains(err.Error(), "device ID missing") {
		t.Fatalf("expected device id missing, got %v", err)
	}

	_, _, _, err = OAuthTokenPassword(context.Background(), foodora.OAuthPasswordRequest{}, PasswordOptions{BaseURL: "://bad", DeviceID: "dev"})
	if err == nil {
		t.Fatalf("expected parse error")
	}

	_, _, _, err = OAuthTokenPassword(context.Background(), foodora.OAuthPasswordRequest{}, PasswordOptions{BaseURL: "https://", DeviceID: "dev"})
	if err == nil || !strings.Contains(err.Error(), "host missing") {
		t.Fatalf("expected host missing, got %v", err)
	}
}

func TestNewOAuthTokenURL(t *testing.T) {
	if got := newOAuthTokenURL("https://hu.fd-api.com/api/v5"); got != "https://hu.fd-api.com/api/v5/oauth2/token" {
		t.Fatalf("got %q", got)
	}
	if got := newOAuthTokenURL("not a url"); got != "not a url" {
		t.Fatalf("got %q", got)
	}
}

func TestParseMfaTriggered(t *testing.T) {
	body := []byte(`{
  "code": "mfa_triggered",
  "metadata": {
    "more_information": {
      "channel": "sms",
      "email": "p@example.com",
      "mfa_token": "tok"
    }
  }
}`)
	ch, ok := parseMfaTriggered(body, map[string]string{"ratelimit-reset": "13"})
	if !ok {
		t.Fatalf("expected ok")
	}
	if ch.MfaToken != "tok" || ch.Channel != "sms" || ch.Email != "p@example.com" || ch.RateLimitReset != 13 {
		t.Fatalf("unexpected: %#v", ch)
	}
}

func TestOAuthTokenPassword_Success_UsesScriptOutput(t *testing.T) {
	orig := runAuthScript
	defer func() { runAuthScript = orig }()

	runAuthScript = func(ctx context.Context, td, scriptPath, outPath string, input []byte, opts PasswordOptions, timeout time.Duration, playwright string) (scriptOutput, error) {
		return scriptOutput{
			Status:       200,
			Body:         `{"access_token":"a","refresh_token":"r","expires_in":1}`,
			CookieHeader: " c=1 ",
			UserAgent:    " ua ",
		}, nil
	}

	tok, mfa, sess, err := OAuthTokenPassword(context.Background(), foodora.OAuthPasswordRequest{
		Username:     "u",
		Password:     "p",
		ClientSecret: "s",
	}, PasswordOptions{
		BaseURL:  "https://mj.fd-api.com/api/v5/",
		DeviceID: "dev",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if mfa != nil {
		t.Fatalf("unexpected mfa: %#v", mfa)
	}
	if tok.AccessToken != "a" || tok.RefreshToken != "r" {
		t.Fatalf("unexpected tok: %#v", tok)
	}
	if sess.Host != "mj.fd-api.com" || sess.CookieHeader != "c=1" || sess.UserAgent != "ua" {
		t.Fatalf("unexpected sess: %#v", sess)
	}
}

func TestOAuthTokenPassword_MfaTriggered_UsesScriptOutput(t *testing.T) {
	orig := runAuthScript
	defer func() { runAuthScript = orig }()

	runAuthScript = func(ctx context.Context, td, scriptPath, outPath string, input []byte, opts PasswordOptions, timeout time.Duration, playwright string) (scriptOutput, error) {
		return scriptOutput{
			Status: 401,
			Body:   `{"code":"mfa_triggered","metadata":{"more_information":{"channel":"sms","email":"e","mfa_token":"tok"}}}`,
			Headers: map[string]string{
				"ratelimit-reset": "9",
			},
		}, nil
	}

	tok, mfa, _, err := OAuthTokenPassword(context.Background(), foodora.OAuthPasswordRequest{
		Username:     "u",
		Password:     "p",
		ClientSecret: "s",
	}, PasswordOptions{
		BaseURL:  "https://mj.fd-api.com/api/v5/",
		DeviceID: "dev",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if tok.AccessToken != "" {
		t.Fatalf("unexpected tok: %#v", tok)
	}
	if mfa == nil || mfa.MfaToken != "tok" || mfa.RateLimitReset != 9 {
		t.Fatalf("unexpected mfa: %#v", mfa)
	}
}

func TestOAuthTokenPassword_ErrorStatus_ReturnsHTTPError(t *testing.T) {
	orig := runAuthScript
	defer func() { runAuthScript = orig }()

	runAuthScript = func(ctx context.Context, td, scriptPath, outPath string, input []byte, opts PasswordOptions, timeout time.Duration, playwright string) (scriptOutput, error) {
		return scriptOutput{Status: 403, Body: `{"error":"nope"}`}, nil
	}

	_, _, _, err := OAuthTokenPassword(context.Background(), foodora.OAuthPasswordRequest{
		Username:     "u",
		Password:     "p",
		ClientSecret: "s",
	}, PasswordOptions{
		BaseURL:  "https://mj.fd-api.com/api/v5/",
		DeviceID: "dev",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var he *foodora.HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
}

func TestOAuthTokenPassword_RealRunner_WithFakeNodeNpm(t *testing.T) {
	fakeBin := t.TempDir()

	writeExe(t, filepath.Join(fakeBin, "npm"), `#!/bin/sh
set -e
mkdir -p node_modules/.bin
cat > node_modules/.bin/playwright <<'EOF'
#!/bin/sh
exit 0
EOF
chmod +x node_modules/.bin/playwright
exit 0
`)

	writeExe(t, filepath.Join(fakeBin, "node"), `#!/bin/sh
set -e
cat > "$ORDERCLI_OUTPUT_PATH" <<'EOF'
{"status":200,"body":"{\"access_token\":\"a\",\"refresh_token\":\"r\",\"expires_in\":1}","headers":{},"cookie_header":"c=1","user_agent":"ua"}
EOF
exit 0
`)

	withPATH(t, fakeBin)

	tok, mfa, sess, err := OAuthTokenPassword(context.Background(), foodora.OAuthPasswordRequest{
		Username:     "u",
		Password:     "p",
		ClientSecret: "s",
	}, PasswordOptions{
		BaseURL:    "https://mj.fd-api.com/api/v5/",
		DeviceID:   "dev",
		Timeout:    10 * time.Second,
		Playwright: "playwright@0.0.0",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if mfa != nil {
		t.Fatalf("unexpected mfa: %#v", mfa)
	}
	if tok.AccessToken != "a" || sess.CookieHeader != "c=1" || sess.UserAgent != "ua" {
		t.Fatalf("unexpected: tok=%#v sess=%#v", tok, sess)
	}
}

func withPATH(t *testing.T, prefixDir string) {
	t.Helper()
	old := os.Getenv("PATH")
	if err := os.Setenv("PATH", prefixDir+string(os.PathListSeparator)+old); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", old) })
}

func writeExe(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

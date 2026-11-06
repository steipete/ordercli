package foodora

import (
	"net/http"
	"strings"
	"testing"
)

func TestParseMfaTriggered(t *testing.T) {
	body := []byte(`{
  "code": "mfa_triggered",
  "metadata": {
    "more_information": {
      "channel": "sms",
      "email": "peter@example.com",
      "mfa_token": "tok123"
    }
  }
}`)
	h := http.Header{}
	h.Set("ratelimit-reset", "12")

	ch, ok := parseMfaTriggered(body, h)
	if !ok {
		t.Fatalf("expected ok")
	}
	if ch.Channel != "sms" || ch.Email != "peter@example.com" || ch.MfaToken != "tok123" || ch.RateLimitReset != 12 {
		t.Fatalf("unexpected challenge: %#v", ch)
	}
}

func TestRedactSensitive_JSON(t *testing.T) {
	raw := []byte(`{
  "access_token":"a",
  "refresh_token":"r",
  "client_secret":"s",
  "password":"p",
  "otp":"o",
  "address":{"id":"123","street":"Main"},
  "nested":{"mfa_token":"m"}
}`)
	out := redactSensitive(raw)

	for _, leak := range []string{`"a"`, `"r"`, `"s"`, `"p"`, `"o"`, `"m"`, "Main"} {
		if strings.Contains(out, leak) {
			t.Fatalf("redaction leaked %q: %s", leak, out)
		}
	}
	for _, want := range []string{`"access_token":"***"`, `"refresh_token":"***"`, `"client_secret":"***"`, `"password":"***"`, `"otp":"***"`, `"mfa_token":"***"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in: %s", want, out)
		}
	}
	if !strings.Contains(out, `"address":"***"`) {
		t.Fatalf("expected address redaction in: %s", out)
	}
}

func TestHTTPError_NonJSONRedactsSecrets(t *testing.T) {
	body := `oops {"access_token":"tokenSECRET","refresh_token":"refreshSECRET","client_secret":"clientSECRET","password":"pwSECRET","mfa_token":"mfaSECRET","otp":"otpSECRET","x-otp":"xotpSECRET","address":"addrSECRET"} tail`
	spaced := `note "access_token" : "spacedSECRET"`
	for _, raw := range []string{body, spaced} {
		s := (&HTTPError{
			Method:     "GET",
			URL:        "https://example.invalid/orders",
			StatusCode: 500,
			Body:       []byte(raw),
		}).Error()
		for _, leak := range []string{"tokenSECRET", "refreshSECRET", "clientSECRET", "pwSECRET", "mfaSECRET", "otpSECRET", "xotpSECRET", "addrSECRET", "spacedSECRET"} {
			if strings.Contains(s, leak) {
				t.Fatalf("leaked %s in %s", leak, s)
			}
		}
		if !strings.Contains(s, "***") {
			t.Fatalf("expected redaction marker: %s", s)
		}
	}
	s := (&HTTPError{Method: "GET", URL: "https://example.invalid/orders", StatusCode: 500, Body: []byte(body)}).Error()
	t.Log(s)
	for _, want := range []string{`"access_token":"***"`, `"x-otp":"***"`, `"address":"***"`, "oops ", " tail"} {
		if !strings.Contains(s, want) {
			t.Fatalf("expected %q in %s", want, s)
		}
	}
}

func TestHTTPError_Error_Redacts(t *testing.T) {
	e := &HTTPError{
		Method:     "POST",
		URL:        "https://example.invalid",
		StatusCode: 401,
		Body:       []byte(`{"password":"pwSECRET","address":{"street":"streetSECRET"},"access_token":"tokenSECRET"}`),
	}
	s := e.Error()
	if strings.Contains(s, "pwSECRET") || strings.Contains(s, "streetSECRET") || strings.Contains(s, "tokenSECRET") {
		t.Fatalf("leaked sensitive data: %s", s)
	}
	if !strings.Contains(s, "***") {
		t.Fatalf("expected redaction marker: %s", s)
	}
}

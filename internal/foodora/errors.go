package foodora

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type HTTPError struct {
	Method     string
	URL        string
	StatusCode int
	Body       []byte
}

func (e *HTTPError) Error() string {
	body := redactSensitive(e.Body)
	if len(body) > 300 {
		body = body[:300] + "…"
	}
	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.URL, e.StatusCode, body)
}

var sensitiveJSONKeys = map[string]struct{}{
	"access_token":  {},
	"refresh_token": {},
	"client_secret": {},
	"password":      {},
	"mfa_token":     {},
	"otp":           {},
	"x-otp":         {},
}

var sensitiveJSONValueRE = regexp.MustCompile(`(?i)(\"(?:access_token|refresh_token|client_secret|password|mfa_token|otp)\"\\s*:\\s*)\"[^\"]*\"`)

func redactSensitive(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	var v any
	if err := json.Unmarshal(b, &v); err == nil {
		redactAny(v)
		if out, err := json.Marshal(v); err == nil {
			return string(out)
		}
	}

	// Best-effort: redact common JSON patterns in string bodies.
	s := string(b)
	return sensitiveJSONValueRE.ReplaceAllString(s, `$1"***"`)
}

func redactAny(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, vv := range t {
			if _, ok := sensitiveJSONKeys[strings.ToLower(k)]; ok {
				t[k] = "***"
				continue
			}
			redactAny(vv)
		}
	case []any:
		for i := range t {
			redactAny(t[i])
		}
	}
}

type MfaChallenge struct {
	Channel        string
	Email          string
	MfaToken       string
	RateLimitReset int
}

func parseMfaTriggered(body []byte, header http.Header) (MfaChallenge, bool) {
	var raw struct {
		Code     string `json:"code"`
		Metadata struct {
			MoreInformation struct {
				Channel  string `json:"channel"`
				Email    string `json:"email"`
				MfaToken string `json:"mfa_token"`
			} `json:"more_information"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return MfaChallenge{}, false
	}
	if raw.Code != "mfa_triggered" {
		return MfaChallenge{}, false
	}

	reset := 30
	if v := header.Get("ratelimit-reset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			reset = n
		}
	}

	ch := MfaChallenge{
		Channel:        raw.Metadata.MoreInformation.Channel,
		Email:          raw.Metadata.MoreInformation.Email,
		MfaToken:       raw.Metadata.MoreInformation.MfaToken,
		RateLimitReset: reset,
	}
	if ch.MfaToken == "" {
		return MfaChallenge{}, false
	}
	return ch, true
}

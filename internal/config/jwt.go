package config

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

func (c FoodoraConfig) AccessTokenExpiresAt() (time.Time, bool) {
	return AccessTokenExpiresAt(c.AccessToken)
}

func AccessTokenExpiresAt(accessToken string) (time.Time, bool) {
	return jwtExpiry(accessToken)
}

func jwtExpiry(token string) (time.Time, bool) {
	if strings.TrimSpace(token) == "" {
		return time.Time{}, false
	}
	_, payloadB64, ok := strings.Cut(token, ".")
	if !ok {
		return time.Time{}, false
	}
	payloadB64, _, ok = strings.Cut(payloadB64, ".")
	if !ok {
		return time.Time{}, false
	}

	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return time.Time{}, false
	}

	var v struct {
		Exp     int64 `json:"exp"`
		Expires int64 `json:"expires"`
	}
	if err := json.Unmarshal(payload, &v); err != nil {
		return time.Time{}, false
	}
	exp := v.Exp
	if exp <= 0 {
		exp = v.Expires
	}
	if exp <= 0 {
		return time.Time{}, false
	}
	return time.Unix(exp, 0).UTC(), true
}

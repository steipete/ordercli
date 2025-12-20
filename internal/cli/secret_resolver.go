package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/steipete/ordercli/internal/firebase"
)

type resolvedSecret struct {
	Secret     string
	FromEnv    bool
	FromConfig bool
	FromFetch  bool
}

func (s *state) resolveClientSecret(ctx context.Context, clientID string) (resolvedSecret, error) {
	cfg := s.foodora()
	if clientID == "" {
		clientID = strings.TrimSpace(cfg.OAuthClientID)
	}
	if clientID == "" {
		clientID = "android"
	}

	// Only reuse cached secrets when we know which client_id they belong to.
	// Legacy configs may have a stored secret without oauth_client_id; assume that is for android only.
	if cfg.ClientSecret != "" && strings.EqualFold(strings.TrimSpace(cfg.OAuthClientID), clientID) {
		return resolvedSecret{Secret: cfg.ClientSecret, FromConfig: true}, nil
	}
	if cfg.ClientSecret != "" && strings.TrimSpace(cfg.OAuthClientID) == "" && clientID == "android" {
		return resolvedSecret{Secret: cfg.ClientSecret, FromConfig: true}, nil
	}
	if v := os.Getenv("FOODORA_CLIENT_SECRET"); v != "" {
		return resolvedSecret{Secret: v, FromEnv: true}, nil
	}

	secret, err := fetchClientSecretFromRemoteConfig(ctx, s.firebaseConfig(), s.remoteConfigKeyCandidates(), clientID)
	if err != nil {
		return resolvedSecret{}, err
	}
	if secret == "" {
		return resolvedSecret{}, errors.New("fetched empty client secret")
	}

	// Cache for next run.
	cfg.ClientSecret = secret
	cfg.OAuthClientID = clientID
	s.markDirty()
	return resolvedSecret{Secret: secret, FromFetch: true}, nil
}

func (s *state) forceFetchClientSecret(ctx context.Context, clientID string) (resolvedSecret, error) {
	cfg := s.foodora()
	if clientID == "" {
		clientID = strings.TrimSpace(cfg.OAuthClientID)
	}
	if clientID == "" {
		clientID = "android"
	}

	secret, err := fetchClientSecretFromRemoteConfig(ctx, s.firebaseConfig(), s.remoteConfigKeyCandidates(), clientID)
	if err != nil {
		return resolvedSecret{}, err
	}
	if secret == "" {
		return resolvedSecret{}, errors.New("fetched empty client secret")
	}

	cfg.ClientSecret = secret
	cfg.OAuthClientID = clientID
	s.markDirty()
	return resolvedSecret{Secret: secret, FromFetch: true}, nil
}

func (s *state) firebaseConfig() firebase.APKFirebaseConfig {
	cfg := s.foodora()
	if strings.EqualFold(cfg.TargetCountryISO, "AT") {
		return firebase.MjamAT
	}
	if strings.HasPrefix(strings.ToUpper(cfg.GlobalEntityID), "MJM_") {
		return firebase.MjamAT
	}
	if strings.Contains(strings.ToLower(cfg.BaseURL), "mj.fd-api.com") {
		return firebase.MjamAT
	}
	return firebase.NetPincerHU
}

func (s *state) remoteConfigKeyCandidates() []string {
	cfg := s.foodora()
	var keys []string

	if u, err := url.Parse(cfg.BaseURL); err == nil {
		host := strings.ToLower(u.Hostname())
		if strings.HasSuffix(host, ".fd-api.com") {
			if sub := strings.Split(host, ".")[0]; sub != "" {
				keys = append(keys, strings.ToUpper(sub))
			}
		}
	}

	if iso := strings.ToUpper(strings.TrimSpace(cfg.TargetCountryISO)); iso != "" {
		keys = append(keys, iso)
	}

	seen := map[string]bool{}
	var out []string
	for _, k := range keys {
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	if len(out) == 0 {
		return []string{"HU"}
	}
	return out
}

func fetchClientSecretFromRemoteConfig(ctx context.Context, cfg firebase.APKFirebaseConfig, keys []string, clientID string) (string, error) {
	rc := firebase.NewRemoteConfigClient(cfg)
	resp, err := rc.Fetch(ctx)
	if err != nil {
		return "", err
	}

	raw, ok := resp.Entries["client_secrets"]
	if !ok || strings.TrimSpace(raw) == "" {
		return "", errors.New("remote config key client_secrets missing/empty")
	}

	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return "", fmt.Errorf("client_secrets not JSON map: %w", err)
	}

	var rawByCountry string
	for _, k := range keys {
		rawByCountry = strings.TrimSpace(m[strings.ToUpper(k)])
		if rawByCountry != "" {
			break
		}
	}
	if rawByCountry == "" {
		return "", fmt.Errorf("client_secrets missing/empty for keys: %s", strings.Join(keys, ","))
	}

	// Newer configs: per-country JSON blob containing {android: "...", corp_android: "..."}.
	if strings.HasPrefix(rawByCountry, "{") {
		var per map[string]string
		if err := json.Unmarshal([]byte(rawByCountry), &per); err != nil {
			return "", fmt.Errorf("client_secrets entry not JSON map: %w", err)
		}
		if clientID != "" {
			if v := strings.TrimSpace(per[clientID]); v != "" {
				return v, nil
			}
		}
		if v := strings.TrimSpace(per["android"]); v != "" {
			return v, nil
		}
		if v := strings.TrimSpace(per["corp_android"]); v != "" {
			return v, nil
		}
		return "", errors.New("client_secrets entry missing android/corp_android")
	}

	// Older configs: per-country value is the secret itself.
	return rawByCountry, nil
}

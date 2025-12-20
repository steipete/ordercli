package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steipete/ordercli/internal/chromecookies"
	"github.com/steipete/ordercli/internal/foodora"
	"github.com/steipete/ordercli/internal/version"
)

func newSessionCmd(st *state) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Session helpers",
	}
	cmd.AddCommand(newSessionChromeCmd(st))
	cmd.AddCommand(newSessionRefreshCmd(st))
	return cmd
}

func newSessionChromeCmd(st *state) *cobra.Command {
	var profile string
	var cookiePath string
	var timeout time.Duration
	var url string
	var forceClientID string

	cmd := &cobra.Command{
		Use:   "chrome",
		Short: "Import refresh_token (+ device_token) from Chrome cookies",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := st.foodora()
			if cfg.BaseURL == "" {
				return errors.New("missing base_url (run `ordercli foodora config set --country ...`)")
			}
			if strings.TrimSpace(url) == "" {
				u, ok := defaultWebURLForConfig(st)
				if !ok {
					return errors.New("--url required (e.g. https://www.foodora.at/)")
				}
				url = u
			}

			cacheDir := filepath.Join(filepath.Dir(st.configPath), "chrome-cookies")
			res, err := chromecookies.LoadCookieHeader(cmd.Context(), chromecookies.Options{
				TargetURL:          strings.TrimSpace(url),
				ChromeProfile:      profile,
				ExplicitCookiePath: cookiePath,
				FilterNames:        []string{"token", "refresh_token", "device_token"},
				Timeout:            timeout,
				CacheDir:           cacheDir,
				LogWriter:          cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}

			kv := parseCookieHeader(res.CookieHeader)
			access := strings.TrimSpace(kv["token"])
			refresh := strings.TrimSpace(kv["refresh_token"])
			deviceToken := strings.TrimSpace(kv["device_token"])
			if refresh == "" {
				return errors.New("missing refresh_token cookie (are you logged in in Chrome on --url?)")
			}
			if deviceToken == "" {
				return errors.New("missing device_token cookie (try reloading the site in Chrome and retry)")
			}

			if cid := strings.TrimSpace(forceClientID); cid != "" {
				cfg.OAuthClientID = cid
			} else if access != "" {
				if cid, ok := jwtClientID(access); ok {
					cfg.OAuthClientID = cid
				}
			} else if strings.TrimSpace(cfg.OAuthClientID) == "" {
				cfg.OAuthClientID = "android"
			}
			if access != "" {
				if exp, ok := jwtExpiry(access); ok {
					cfg.ExpiresAt = time.Unix(exp, 0)
				} else {
					cfg.ExpiresAt = time.Time{}
				}
			}
			cfg.DeviceID = deviceToken
			cfg.AccessToken = access
			cfg.RefreshToken = refresh
			if access == "" {
				cfg.ExpiresAt = time.Time{}
			}
			st.markDirty()

			if access == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "ok (refresh_token imported; run `ordercli foodora session refresh`)")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "site URL that holds the cookies (e.g. https://www.foodora.at/)")
	cmd.Flags().StringVar(&profile, "profile", "", "Chrome profile name (Default, Profile 1, ...) or path to profile dir")
	cmd.Flags().StringVar(&cookiePath, "cookie-path", "", "explicit Cookies DB path or profile dir (overrides --profile)")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "cookie read timeout (keychain prompts may need longer)")
	cmd.Flags().StringVar(&forceClientID, "client-id", "", "override oauth client_id to pair with the refresh token (default: from JWT)")
	return cmd
}

func newSessionRefreshCmd(st *state) *cobra.Command {
	var forceClientID string

	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh access token (uses stored refresh_token)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := st.foodora()
			if cfg.BaseURL == "" {
				return errors.New("missing base_url (run `ordercli foodora config set --country ...`)")
			}
			if cfg.RefreshToken == "" {
				return errors.New("missing refresh_token (run `ordercli foodora login ...` or `ordercli foodora session chrome ...`)")
			}

			clientID := strings.TrimSpace(forceClientID)
			if clientID == "" {
				clientID = strings.TrimSpace(cfg.OAuthClientID)
			}
			if clientID == "" {
				if cid, ok := jwtClientID(cfg.AccessToken); ok {
					clientID = cid
				}
			}
			if clientID == "" {
				clientID = "android"
			}

			sec, err := st.resolveClientSecret(cmd.Context(), clientID)
			if err != nil {
				return err
			}

			_, cookie := st.cookieHeaderForBaseURL()
			prof := st.appHeaders()
			ua := cfg.HTTPUserAgent
			if ua == "" && prof.UserAgent != "" {
				ua = prof.UserAgent
			}
			if ua == "" {
				ua = "ordercli/" + version.Version
			}

			c, err := foodora.New(foodora.Options{
				BaseURL:          cfg.BaseURL,
				DeviceID:         cfg.DeviceID,
				GlobalEntityID:   cfg.GlobalEntityID,
				TargetCountryISO: cfg.TargetCountryISO,
				UserAgent:        ua,
				CookieHeader:     cookie,
				FPAPIKey:         prof.FPAPIKey,
				AppName:          prof.AppName,
				OriginalUserAgent: func() string {
					if strings.HasPrefix(ua, "Android-app-") {
						return ua
					}
					return ""
				}(),
			})
			if err != nil {
				return err
			}

			now := time.Now()
			tok, err := c.OAuthTokenRefresh(cmd.Context(), foodora.OAuthRefreshRequest{
				RefreshToken: cfg.RefreshToken,
				ClientSecret: sec.Secret,
				ClientID:     clientID,
			})
			if err != nil && isInvalidClientErr(err) {
				if sec2, ferr := st.forceFetchClientSecret(cmd.Context(), clientID); ferr == nil {
					tok, err = c.OAuthTokenRefresh(cmd.Context(), foodora.OAuthRefreshRequest{
						RefreshToken: cfg.RefreshToken,
						ClientSecret: sec2.Secret,
						ClientID:     clientID,
					})
				}
			}
			if err != nil {
				return err
			}

			cfg.AccessToken = tok.AccessToken
			if tok.RefreshToken != "" {
				cfg.RefreshToken = tok.RefreshToken
			}
			cfg.ExpiresAt = tok.ExpiresAt(now)
			if cfg.ExpiresAt.IsZero() {
				if exp, ok := cfg.AccessTokenExpiresAt(); ok {
					cfg.ExpiresAt = exp
				}
			}
			cfg.OAuthClientID = clientID
			st.markDirty()
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}

	cmd.Flags().StringVar(&forceClientID, "client-id", "", "override oauth client_id (default: config/JWT)")
	return cmd
}

func defaultWebURLForConfig(st *state) (string, bool) {
	if strings.EqualFold(st.foodora().TargetCountryISO, "AT") {
		return "https://www.foodora.at/", true
	}
	return "", false
}

func parseCookieHeader(header string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(header, ";") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

func jwtExpiry(token string) (int64, bool) {
	_, payloadB64, ok := strings.Cut(token, ".")
	if !ok {
		return 0, false
	}
	payloadB64, _, ok = strings.Cut(payloadB64, ".")
	if !ok {
		return 0, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return 0, false
	}
	var v struct {
		Exp     int64 `json:"exp"`
		Expires int64 `json:"expires"`
	}
	if err := json.Unmarshal(payload, &v); err != nil {
		return 0, false
	}
	exp := v.Exp
	if exp <= 0 {
		exp = v.Expires
	}
	if exp <= 0 {
		return 0, false
	}
	return exp, true
}

func jwtClientID(token string) (string, bool) {
	_, payloadB64, ok := strings.Cut(token, ".")
	if !ok {
		return "", false
	}
	payloadB64, _, ok = strings.Cut(payloadB64, ".")
	if !ok {
		return "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", false
	}
	var v struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(payload, &v); err != nil {
		return "", false
	}
	if strings.TrimSpace(v.ClientID) == "" {
		return "", false
	}
	return strings.TrimSpace(v.ClientID), true
}

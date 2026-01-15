package config

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Version   int       `json:"version"`
	Providers Providers `json:"providers,omitempty"`
}

type Providers struct {
	Foodora   *FoodoraConfig   `json:"foodora,omitempty"`
	Deliveroo *DeliverooConfig `json:"deliveroo,omitempty"`
	Glovo     *GlovoConfig     `json:"glovo,omitempty"`
}

type FoodoraConfig struct {
	BaseURL          string    `json:"base_url"`
	GlobalEntityID   string    `json:"global_entity_id,omitempty"`
	TargetCountryISO string    `json:"target_country_iso,omitempty"`
	DeviceID         string    `json:"device_id"`
	AccessToken      string    `json:"access_token,omitempty"`
	RefreshToken     string    `json:"refresh_token,omitempty"`
	ExpiresAt        time.Time `json:"expires_at,omitempty"`
	ClientSecret     string    `json:"client_secret,omitempty"`
	OAuthClientID    string    `json:"oauth_client_id,omitempty"`

	HTTPUserAgent string            `json:"http_user_agent,omitempty"`
	CookiesByHost map[string]string `json:"cookies_by_host,omitempty"`

	PendingMfaToken     string    `json:"pending_mfa_token,omitempty"`
	PendingMfaChannel   string    `json:"pending_mfa_channel,omitempty"`
	PendingMfaEmail     string    `json:"pending_mfa_email,omitempty"`
	PendingMfaCreatedAt time.Time `json:"pending_mfa_created_at,omitempty"`
}

type DeliverooConfig struct {
	Market  string `json:"market,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
}

type GlovoConfig struct {
	BaseURL     string  `json:"base_url,omitempty"`
	AccessToken string  `json:"access_token,omitempty"`
	DeviceURN   string  `json:"device_urn,omitempty"`
	CityCode    string  `json:"city_code,omitempty"`
	CountryCode string  `json:"country_code,omitempty"`
	Language    string  `json:"language,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ordercli", "config.json"), nil
}

func LegacyPathFoodcli() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "foodcli", "config.json"), nil
}

func LegacyPathFoodoracli() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "foodoracli", "config.json"), nil
}

func Load(path string) (Config, error) {
	var cfg Config

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return New(), nil
		}
		return cfg, err
	}

	var sniff struct {
		Providers json.RawMessage `json:"providers"`
	}
	if err := json.Unmarshal(b, &sniff); err != nil {
		return cfg, err
	}

	if len(sniff.Providers) > 0 {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return cfg, err
		}
	} else {
		var legacy FoodoraConfig
		if err := json.Unmarshal(b, &legacy); err != nil {
			return cfg, err
		}
		cfg = New()
		cfg.Providers.Foodora = &legacy
	}

	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Providers.Foodora != nil && cfg.Providers.Foodora.DeviceID == "" {
		cfg.Providers.Foodora.DeviceID = newDeviceID()
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Providers.Foodora != nil && cfg.Providers.Foodora.DeviceID == "" {
		cfg.Providers.Foodora.DeviceID = newDeviceID()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func New() Config {
	return Config{
		Version: 1,
	}
}

func (c *Config) Foodora() *FoodoraConfig {
	if c.Providers.Foodora == nil {
		c.Providers.Foodora = &FoodoraConfig{}
	}
	if c.Providers.Foodora.DeviceID == "" {
		c.Providers.Foodora.DeviceID = newDeviceID()
	}
	return c.Providers.Foodora
}

func (c *Config) Deliveroo() *DeliverooConfig {
	if c.Providers.Deliveroo == nil {
		c.Providers.Deliveroo = &DeliverooConfig{}
	}
	return c.Providers.Deliveroo
}

func (c *Config) Glovo() *GlovoConfig {
	if c.Providers.Glovo == nil {
		c.Providers.Glovo = &GlovoConfig{
			BaseURL: "https://api.glovoapp.com",
		}
	}
	return c.Providers.Glovo
}

func (c FoodoraConfig) HasSession() bool {
	return c.AccessToken != "" && c.RefreshToken != ""
}

func (c FoodoraConfig) TokenLikelyExpired(now time.Time) bool {
	if c.AccessToken == "" {
		return true
	}
	if c.ExpiresAt.IsZero() {
		if exp, ok := c.AccessTokenExpiresAt(); ok {
			return !exp.After(now.Add(30 * time.Second))
		}
		return false
	}
	return !c.ExpiresAt.After(now.Add(30 * time.Second))
}

func newDeviceID() string {
	// UUIDv4-ish; good enough for header X-Device.
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("dev-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	)
}

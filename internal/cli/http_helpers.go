package cli

import (
	"net/url"
	"strings"
)

type appHeaderProfile struct {
	FPAPIKey  string
	AppName   string
	UserAgent string
}

func cookieHost(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func (s *state) cookieHeaderForBaseURL() (host string, cookie string) {
	cfg := s.foodora()
	host = cookieHost(cfg.BaseURL)
	if host == "" || len(cfg.CookiesByHost) == 0 {
		return host, ""
	}
	return host, strings.TrimSpace(cfg.CookiesByHost[host])
}

func (s *state) appHeaders() appHeaderProfile {
	cfg := s.foodora()
	p := appHeaderProfile{
		FPAPIKey: "android",
	}
	if strings.EqualFold(cfg.TargetCountryISO, "AT") || strings.HasPrefix(strings.ToUpper(cfg.GlobalEntityID), "MJM_") || strings.Contains(strings.ToLower(cfg.BaseURL), "mj.fd-api.com") {
		p.AppName = "at.mjam"
		// From the provided at.mjam APKM (v25.3.0 / build 250300134).
		p.UserAgent = "Android-app-25.3.0(250300134)"
	}
	return p
}

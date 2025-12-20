package browserauth

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/steipete/ordercli/internal/foodora"
)

//go:embed login.mjs
var loginScript []byte

type Session struct {
	Host         string
	CookieHeader string
	UserAgent    string
}

type PasswordOptions struct {
	BaseURL    string
	DeviceID   string
	Timeout    time.Duration
	LogWriter  io.Writer
	Playwright string
	ProfileDir string
}

func OAuthTokenPassword(ctx context.Context, req foodora.OAuthPasswordRequest, opts PasswordOptions) (foodora.AuthToken, *foodora.MfaChallenge, Session, error) {
	if opts.BaseURL == "" {
		return foodora.AuthToken{}, nil, Session{}, errors.New("browserauth: base URL missing")
	}
	if opts.DeviceID == "" {
		return foodora.AuthToken{}, nil, Session{}, errors.New("browserauth: device ID missing")
	}

	base, err := url.Parse(opts.BaseURL)
	if err != nil {
		return foodora.AuthToken{}, nil, Session{}, err
	}
	host := base.Hostname()
	if host == "" {
		return foodora.AuthToken{}, nil, Session{}, errors.New("browserauth: base URL host missing")
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	pw := opts.Playwright
	if pw == "" {
		pw = "playwright@1.50.0"
	}

	if _, err := exec.LookPath("node"); err != nil {
		return foodora.AuthToken{}, nil, Session{}, errors.New("browserauth: node not found")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		return foodora.AuthToken{}, nil, Session{}, errors.New("browserauth: npm not found")
	}

	td, err := os.MkdirTemp("", "ordercli-browserauth-*")
	if err != nil {
		return foodora.AuthToken{}, nil, Session{}, err
	}
	defer func() { _ = os.RemoveAll(td) }()

	scriptPath := filepath.Join(td, "login.mjs")
	if err := os.WriteFile(scriptPath, loginScript, 0o600); err != nil {
		return foodora.AuthToken{}, nil, Session{}, err
	}
	outPath := filepath.Join(td, "out.json")

	in := scriptInput{
		BaseURL:       opts.BaseURL,
		DeviceID:      opts.DeviceID,
		Email:         req.Username,
		Password:      req.Password,
		ClientSecret:  req.ClientSecret,
		ClientID:      req.ClientID,
		OTPMethod:     req.OTPMethod,
		OTPCode:       req.OTPCode,
		MfaToken:      req.MfaToken,
		TimeoutMillis: int(timeout.Milliseconds()),
		ProfileDir:    strings.TrimSpace(opts.ProfileDir),
	}
	b, _ := json.Marshal(in)

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	install := exec.CommandContext(cmdCtx, "npm", "install", "--silent", "--no-progress", "--no-fund", "--no-audit", pw) //nolint:gosec
	install.Dir = td
	install.Stdout = io.Discard
	if opts.LogWriter != nil {
		install.Stderr = opts.LogWriter
	} else {
		install.Stderr = io.Discard
	}
	install.Env = append(os.Environ(),
		"npm_config_loglevel=error",
	)
	if err := install.Run(); err != nil {
		return foodora.AuthToken{}, nil, Session{}, fmt.Errorf("browserauth: npm install %s: %w", pw, err)
	}

	playwrightBin := filepath.Join(td, "node_modules", ".bin", "playwright")
	if runtime.GOOS == "windows" {
		playwrightBin += ".cmd"
	}
	installBrowsers := exec.CommandContext(cmdCtx, playwrightBin, "install", "chromium") //nolint:gosec
	installBrowsers.Dir = td
	installBrowsers.Stdout = io.Discard
	if opts.LogWriter != nil {
		installBrowsers.Stderr = opts.LogWriter
	} else {
		installBrowsers.Stderr = io.Discard
	}
	installBrowsers.Env = append(os.Environ(),
		"npm_config_loglevel=error",
	)
	if err := installBrowsers.Run(); err != nil {
		return foodora.AuthToken{}, nil, Session{}, fmt.Errorf("browserauth: playwright install chromium: %w", err)
	}

	cmd := exec.CommandContext(cmdCtx, "node", scriptPath) //nolint:gosec
	cmd.Dir = td
	cmd.Env = append(os.Environ(),
		"ORDERCLI_OUTPUT_PATH="+outPath,
		"FOODCLI_OUTPUT_PATH="+outPath,
		"FOODORACLI_OUTPUT_PATH="+outPath, // legacy
		"npm_config_loglevel=error",
	)
	cmd.Stdin = bytes.NewReader(b)
	cmd.Stdout = io.Discard
	if opts.LogWriter != nil {
		cmd.Stderr = opts.LogWriter
	} else {
		cmd.Stderr = io.Discard
	}

	if err := cmd.Run(); err != nil {
		return foodora.AuthToken{}, nil, Session{}, fmt.Errorf("browserauth: node run: %w", err)
	}

	ob, err := os.ReadFile(outPath)
	if err != nil {
		return foodora.AuthToken{}, nil, Session{}, fmt.Errorf("browserauth: missing output: %w", err)
	}

	var out scriptOutput
	if err := json.Unmarshal(ob, &out); err != nil {
		return foodora.AuthToken{}, nil, Session{}, fmt.Errorf("browserauth: decode output: %w", err)
	}

	sess := Session{
		Host:         host,
		CookieHeader: strings.TrimSpace(out.CookieHeader),
		UserAgent:    strings.TrimSpace(out.UserAgent),
	}

	body := []byte(out.Body)
	if out.Status >= 200 && out.Status < 300 {
		var tok foodora.AuthToken
		if err := json.Unmarshal(body, &tok); err != nil {
			return foodora.AuthToken{}, nil, sess, fmt.Errorf("browserauth: oauth2/token decode: %w", err)
		}
		if tok.AccessToken == "" {
			return foodora.AuthToken{}, nil, sess, fmt.Errorf("browserauth: oauth2/token missing access_token (status %d)", out.Status)
		}
		return tok, nil, sess, nil
	}

	if ch, ok := parseMfaTriggered(body, out.Headers); ok {
		return foodora.AuthToken{}, &ch, sess, nil
	}

	return foodora.AuthToken{}, nil, sess, &foodora.HTTPError{
		Method:     http.MethodPost,
		URL:        newOAuthTokenURL(opts.BaseURL),
		StatusCode: out.Status,
		Body:       body,
	}
}

type scriptInput struct {
	BaseURL       string `json:"base_url"`
	DeviceID      string `json:"device_id"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	ClientSecret  string `json:"client_secret"`
	ClientID      string `json:"client_id"`
	OTPMethod     string `json:"otp_method"`
	OTPCode       string `json:"otp_code"`
	MfaToken      string `json:"mfa_token"`
	TimeoutMillis int    `json:"timeout_millis"`
	ProfileDir    string `json:"profile_dir"`
}

type scriptOutput struct {
	Status       int               `json:"status"`
	Body         string            `json:"body"`
	Headers      map[string]string `json:"headers"`
	CookieHeader string            `json:"cookie_header"`
	UserAgent    string            `json:"user_agent"`
}

func newOAuthTokenURL(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	u = u.ResolveReference(&url.URL{Path: "oauth2/token"})
	return u.String()
}

func parseMfaTriggered(body []byte, headers map[string]string) (foodora.MfaChallenge, bool) {
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
		return foodora.MfaChallenge{}, false
	}
	if raw.Code != "mfa_triggered" {
		return foodora.MfaChallenge{}, false
	}

	reset := 30
	if v := headers["ratelimit-reset"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			reset = n
		}
	}

	ch := foodora.MfaChallenge{
		Channel:        raw.Metadata.MoreInformation.Channel,
		Email:          raw.Metadata.MoreInformation.Email,
		MfaToken:       raw.Metadata.MoreInformation.MfaToken,
		RateLimitReset: reset,
	}
	if ch.MfaToken == "" {
		return foodora.MfaChallenge{}, false
	}
	return ch, true
}

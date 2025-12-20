package chromecookies

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

//go:embed load.mjs
var loadScript []byte

type Options struct {
	TargetURL          string
	ChromeProfile      string
	ExplicitCookiePath string
	FilterNames        []string
	Timeout            time.Duration
	CacheDir           string
	LogWriter          io.Writer
}

type Result struct {
	CookieHeader string
	CookieCount  int
}

func LoadCookieHeader(ctx context.Context, opts Options) (Result, error) {
	if opts.TargetURL == "" {
		return Result{}, errors.New("chromecookies: TargetURL missing")
	}
	if opts.CacheDir == "" {
		return Result{}, errors.New("chromecookies: CacheDir missing")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}

	if _, err := exec.LookPath("node"); err != nil {
		return Result{}, errors.New("chromecookies: node not found")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		return Result{}, errors.New("chromecookies: npm not found")
	}

	if err := os.MkdirAll(opts.CacheDir, 0o755); err != nil {
		return Result{}, err
	}
	if err := ensureNpmProject(ctx, opts.CacheDir, opts.LogWriter); err != nil {
		return Result{}, err
	}

	scriptPath := filepath.Join(opts.CacheDir, "load.mjs")
	if err := os.WriteFile(scriptPath, loadScript, 0o600); err != nil {
		return Result{}, err
	}

	td, err := os.MkdirTemp("", "ordercli-chromecookies-*")
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = os.RemoveAll(td) }()

	outPath := filepath.Join(td, "out.json")

	in := scriptInput{
		TargetURL:          opts.TargetURL,
		ChromeProfile:      opts.ChromeProfile,
		ExplicitCookiePath: opts.ExplicitCookiePath,
		FilterNames:        opts.FilterNames,
		TimeoutMillis:      int(opts.Timeout.Milliseconds()),
	}
	b, _ := json.Marshal(in)

	cmdCtx, cancel := context.WithTimeout(ctx, opts.Timeout+5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "node", scriptPath) //nolint:gosec
	cmd.Dir = opts.CacheDir
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
		// Best-effort: the script may have written a structured error before exiting non-zero.
		if ob, readErr := os.ReadFile(outPath); readErr == nil {
			var out scriptOutput
			if jsonErr := json.Unmarshal(ob, &out); jsonErr == nil && out.Error != "" {
				return Result{}, errors.New(out.Error)
			}
		}
		return Result{}, fmt.Errorf("chromecookies: node run: %w", err)
	}

	ob, err := os.ReadFile(outPath)
	if err != nil {
		return Result{}, fmt.Errorf("chromecookies: missing output: %w", err)
	}

	var out scriptOutput
	if err := json.Unmarshal(ob, &out); err != nil {
		return Result{}, fmt.Errorf("chromecookies: decode output: %w", err)
	}
	if out.Error != "" {
		return Result{}, errors.New(out.Error)
	}
	return Result{
		CookieHeader: out.CookieHeader,
		CookieCount:  out.CookieCount,
	}, nil
}

type scriptInput struct {
	TargetURL          string   `json:"target_url"`
	ChromeProfile      string   `json:"chrome_profile"`
	ExplicitCookiePath string   `json:"explicit_cookie_path"`
	FilterNames        []string `json:"filter_names"`
	TimeoutMillis      int      `json:"timeout_millis"`
}

type scriptOutput struct {
	CookieHeader string `json:"cookie_header"`
	CookieCount  int    `json:"cookie_count"`
	Error        string `json:"error"`
}

func ensureNpmProject(ctx context.Context, dir string, logWriter io.Writer) error {
	nodeModules := filepath.Join(dir, "node_modules", "chrome-cookies-secure", "package.json")
	if _, err := os.Stat(nodeModules); err == nil {
		return nil
	}

	pkg := []byte("{\"private\":true,\"type\":\"module\",\"dependencies\":{\"chrome-cookies-secure\":\"3.0.0\"}}\n")
	if err := os.WriteFile(filepath.Join(dir, "package.json"), pkg, 0o600); err != nil {
		return err
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	install := exec.CommandContext(cmdCtx, "npm", "install", "--silent", "--no-progress", "--no-fund", "--no-audit") //nolint:gosec
	install.Dir = dir
	install.Stdout = io.Discard
	if logWriter != nil {
		install.Stderr = logWriter
	} else {
		install.Stderr = io.Discard
	}
	install.Env = append(os.Environ(),
		"npm_config_loglevel=error",
	)
	if err := install.Run(); err != nil {
		return fmt.Errorf("chromecookies: npm install chrome-cookies-secure: %w", err)
	}
	return nil
}

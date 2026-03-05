package browserpage

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
	"runtime"
	"strings"
	"time"
)

//go:embed fetch.mjs
var fetchScript []byte

type Options struct {
	Timeout    time.Duration
	Headless   bool
	LogWriter  io.Writer
	Playwright string
}

type Result struct {
	FinalURL string `json:"final_url"`
	Title    string `json:"title"`
	Text     string `json:"text"`
}

type scriptInput struct {
	URL           string `json:"url"`
	TimeoutMillis int    `json:"timeout_millis"`
	Headless      bool   `json:"headless"`
}

var runFetchScriptFunc = runFetchScript

func ReadText(ctx context.Context, targetURL string, opts Options) (Result, error) {
	targetURL = strings.TrimSpace(targetURL)
	if targetURL == "" {
		return Result{}, errors.New("browserpage: url missing")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Minute
	}
	pw := strings.TrimSpace(opts.Playwright)
	if pw == "" {
		pw = "playwright@1.58.2"
	}

	td, err := os.MkdirTemp("", "ordercli-browserpage-*")
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = os.RemoveAll(td) }()

	scriptPath := filepath.Join(td, "fetch.mjs")
	if err := os.WriteFile(scriptPath, fetchScript, 0o600); err != nil {
		return Result{}, err
	}
	outPath := filepath.Join(td, "out.json")

	in := scriptInput{
		URL:           targetURL,
		TimeoutMillis: int(opts.Timeout.Milliseconds()),
		Headless:      opts.Headless,
	}
	b, _ := json.Marshal(in)

	out, err := runFetchScriptFunc(ctx, td, scriptPath, outPath, b, opts, pw)
	if err != nil {
		return Result{}, err
	}

	var res Result
	if err := json.Unmarshal(out, &res); err != nil {
		return Result{}, fmt.Errorf("browserpage: decode output: %w", err)
	}
	return res, nil
}

func runFetchScript(ctx context.Context, td, scriptPath, outPath string, input []byte, opts Options, playwright string) ([]byte, error) {
	if _, err := exec.LookPath("node"); err != nil {
		return nil, errors.New("browserpage: node not found")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		return nil, errors.New("browserpage: npm not found")
	}

	cmdCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	install := exec.CommandContext(cmdCtx, "npm", "install", "--silent", "--no-progress", "--no-fund", "--no-audit", playwright) //nolint:gosec
	install.Dir = td
	install.Stdout = io.Discard
	if opts.LogWriter != nil {
		install.Stderr = opts.LogWriter
	} else {
		install.Stderr = io.Discard
	}
	install.Env = append(os.Environ(), "npm_config_loglevel=error")
	if err := install.Run(); err != nil {
		return nil, fmt.Errorf("browserpage: npm install %s: %w", playwright, err)
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
	installBrowsers.Env = append(os.Environ(), "npm_config_loglevel=error")
	if err := installBrowsers.Run(); err != nil {
		return nil, fmt.Errorf("browserpage: playwright install chromium: %w", err)
	}

	cmd := exec.CommandContext(cmdCtx, "node", scriptPath) //nolint:gosec
	cmd.Dir = td
	cmd.Env = append(os.Environ(),
		"ORDERCLI_OUTPUT_PATH="+outPath,
		"npm_config_loglevel=error",
	)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = io.Discard
	if opts.LogWriter != nil {
		cmd.Stderr = opts.LogWriter
	} else {
		cmd.Stderr = io.Discard
	}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("browserpage: node run: %w", err)
	}

	out, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("browserpage: missing output: %w", err)
	}
	return out, nil
}

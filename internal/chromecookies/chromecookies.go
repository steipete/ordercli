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
	"strings"
	"time"
)

//go:embed load.mjs
var loadScript []byte

// Keep the install-time overrides until chrome-cookies-secure drops its vulnerable build chain.
const (
	npmPackageJSON = `{"private":true,"type":"module","dependencies":{"chrome-cookies-secure":"3.0.2"},"overrides":{"tar":"7.5.20","@tootallnate/once":"2.0.1"}}`
	npmStateFile   = ".ordercli-dependencies"
)

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

	out, err := runScript(ctx, opts.CacheDir, scriptPath, outPath, b, opts.LogWriter, opts.Timeout)
	if err != nil {
		return Result{}, err
	}
	if out.Error != "" {
		return Result{}, errors.New(out.Error)
	}
	return Result{CookieHeader: out.CookieHeader, CookieCount: out.CookieCount}, nil
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

var runScript = runScriptReal

func ensureNpmProject(ctx context.Context, dir string, logWriter io.Writer) error {
	if npmProjectCurrent(dir) {
		return nil
	}

	pkg := []byte(npmPackageJSON + "\n")
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
	install.Env = append(
		os.Environ(),
		"npm_config_loglevel=error",
	)
	if err := install.Run(); err != nil {
		return fmt.Errorf("chromecookies: npm install chrome-cookies-secure: %w", err)
	}
	if err := verifyInstalledNpmDependencies(dir); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, npmStateFile), []byte(npmPackageJSON+"\n"), 0o600); err != nil {
		return fmt.Errorf("chromecookies: write dependency state: %w", err)
	}
	return nil
}

func npmProjectCurrent(dir string) bool {
	pkg, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil || strings.TrimSpace(string(pkg)) != npmPackageJSON {
		return false
	}
	state, err := os.ReadFile(filepath.Join(dir, npmStateFile))
	if err != nil || strings.TrimSpace(string(state)) != npmPackageJSON {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, "node_modules", "chrome-cookies-secure", "package.json"))
	return err == nil
}

func verifyInstalledNpmDependencies(dir string) error {
	required := map[string]string{
		"chrome-cookies-secure": "3.0.2",
		"tar":                   "7.5.20",
		"@tootallnate/once":     "2.0.1",
	}
	seen := make(map[string]bool, len(required))
	err := filepath.WalkDir(filepath.Join(dir, "node_modules"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "package.json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var pkg struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil {
			return nil
		}
		want, ok := required[pkg.Name]
		if !ok {
			return nil
		}
		if pkg.Version != want {
			return fmt.Errorf("chromecookies: installed %s version %s, want %s", pkg.Name, pkg.Version, want)
		}
		seen[pkg.Name] = true
		return nil
	})
	if err != nil {
		return fmt.Errorf("chromecookies: verify npm dependencies: %w", err)
	}
	for name := range required {
		if !seen[name] {
			return fmt.Errorf("chromecookies: installed dependency %s missing", name)
		}
	}
	return nil
}

func runScriptReal(ctx context.Context, cacheDir, scriptPath, outPath string, input []byte, logWriter io.Writer, timeout time.Duration) (scriptOutput, error) {
	if _, err := exec.LookPath("node"); err != nil {
		return scriptOutput{}, errors.New("chromecookies: node not found")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		return scriptOutput{}, errors.New("chromecookies: npm not found")
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout+5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "node", scriptPath) //nolint:gosec
	cmd.Dir = cacheDir
	cmd.Env = append(
		os.Environ(),
		"ORDERCLI_OUTPUT_PATH="+outPath,
		"FOODCLI_OUTPUT_PATH="+outPath,
		"FOODORACLI_OUTPUT_PATH="+outPath, // legacy
		"npm_config_loglevel=error",
	)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = io.Discard
	if logWriter != nil {
		cmd.Stderr = logWriter
	} else {
		cmd.Stderr = io.Discard
	}

	if err := cmd.Run(); err != nil {
		// Best-effort: the script may have written a structured error before exiting non-zero.
		if ob, readErr := os.ReadFile(outPath); readErr == nil {
			var out scriptOutput
			if jsonErr := json.Unmarshal(ob, &out); jsonErr == nil && out.Error != "" {
				return out, nil
			}
		}
		return scriptOutput{}, fmt.Errorf("chromecookies: node run: %w", err)
	}

	ob, err := os.ReadFile(outPath)
	if err != nil {
		return scriptOutput{}, fmt.Errorf("chromecookies: missing output: %w", err)
	}

	var out scriptOutput
	if err := json.Unmarshal(ob, &out); err != nil {
		return scriptOutput{}, fmt.Errorf("chromecookies: decode output: %w", err)
	}
	return out, nil
}

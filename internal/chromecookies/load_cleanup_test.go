package chromecookies

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLoadScriptRemovesCookieCopy(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}

	t.Run("success", func(t *testing.T) {
		assertLoadScriptCookieCopy(t, "ok")
	})
	t.Run("error", func(t *testing.T) {
		assertLoadScriptCookieCopy(t, "error")
	})
	t.Run("unreadable", func(t *testing.T) {
		if os.Geteuid() == 0 || runtime.GOOS == "windows" {
			t.Skip("requires Unix permission semantics and a non-root user")
		}
		assertLoadScriptLeavesUnreadableSource(t)
	})
}

func assertLoadScriptCookieCopy(t *testing.T, mode string) {
	t.Helper()

	parent := t.TempDir()
	root := t.TempDir()
	fixture := filepath.Join(root, "fixture-Cookies")
	if err := os.WriteFile(fixture, []byte("dummy-cookies-db"), 0o600); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, "seen.txt")
	outPath := filepath.Join(root, "out.json")
	script := writeLoadScriptFixture(t, root)

	runLoadScript(t, script, parent, fixture, outPath, marker, mode)

	markerText, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("cookie module did not see the copy: %v", err)
	}
	if !strings.Contains(string(markerText), "saw_copy=yes") {
		t.Fatalf("marker = %q", markerText)
	}
	if got := cookieCopyDirs(t, parent); len(got) != 0 {
		t.Fatalf("cookie copy still present: %v", got)
	}

	var out scriptOutput
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	switch mode {
	case "ok":
		if out.Error != "" || out.CookieCount != 1 || out.CookieHeader != "fixture=1" {
			t.Fatalf("unexpected output: %#v", out)
		}
	case "error":
		if !strings.Contains(out.Error, "fixture read failed") {
			t.Fatalf("unexpected output: %#v", out)
		}
	default:
		t.Fatalf("unknown mode %s", mode)
	}
}

func assertLoadScriptLeavesUnreadableSource(t *testing.T) {
	t.Helper()

	parent := t.TempDir()
	root := t.TempDir()
	profile := filepath.Join(root, "profile")
	if err := os.MkdirAll(profile, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(profile, "Cookies")
	if err := os.WriteFile(fixture, []byte("dummy-cookies-db"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(fixture, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(fixture, 0o600) })

	marker := filepath.Join(root, "seen.txt")
	outPath := filepath.Join(root, "out.json")
	script := writeLoadScriptFixture(t, root)

	runLoadScript(t, script, parent, fixture, outPath, marker, "error")

	if _, err := os.Stat(fixture); err != nil {
		t.Fatalf("source Cookies file was removed: %v", err)
	}
	if _, err := os.Stat(profile); err != nil {
		t.Fatalf("source profile was removed: %v", err)
	}
	markerText, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markerText), "unreadable") {
		t.Fatalf("marker = %q", markerText)
	}
	if got := cookieCopyDirs(t, parent); len(got) != 0 {
		t.Fatalf("empty cookie copy still present: %v", got)
	}
}

func writeLoadScriptFixture(t *testing.T, root string) string {
	t.Helper()

	modDir := filepath.Join(root, "node_modules", "chrome-cookies-secure")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := "{\"name\":\"chrome-cookies-secure\",\"version\":\"3.0.2\",\"main\":\"index.js\"}\n"
	if err := os.WriteFile(filepath.Join(modDir, "package.json"), []byte(pkg), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "index.js"), []byte(loadScriptTestModule), 0o600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "load.mjs")
	if err := os.WriteFile(script, loadScript, 0o600); err != nil {
		t.Fatal(err)
	}
	return script
}

func runLoadScript(t *testing.T, script, parent, fixture, outPath, marker, mode string) {
	t.Helper()

	input, err := json.Marshal(map[string]any{
		"target_url":           "https://www.foodora.at/",
		"explicit_cookie_path": fixture,
		"timeout_millis":       5000,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", script)
	cmd.Dir = filepath.Dir(script)
	cmd.Stdin = bytes.NewReader(input)
	var logs bytes.Buffer
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	cmd.Env = append(os.Environ(),
		"TMPDIR="+parent,
		"ORDERCLI_OUTPUT_PATH="+outPath,
		"ORDERCLI_LOAD_TEST_MODE="+mode,
		"ORDERCLI_LOAD_TEST_MARKER="+marker,
	)
	if err := cmd.Run(); err != nil && mode == "ok" {
		t.Fatalf("node: %v\n%s", err, logs.String())
	}
}

func cookieCopyDirs(t *testing.T, parent string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(parent, "ordercli-cookies-secure-*"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

const loadScriptTestModule = `const fs = require('fs');
const path = require('path');

function getCookiesPromised(url, format, profile) {
  const cookies = path.join(profile, 'Cookies');
  let saw = 'missing';
  try {
    fs.accessSync(cookies, fs.constants.R_OK);
    const st = fs.statSync(cookies);
    saw = 'saw_copy=yes bytes=' + st.size;
  } catch {
    saw = 'unreadable';
  }
  fs.writeFileSync(process.env.ORDERCLI_LOAD_TEST_MARKER, saw + '\n');
  if (saw === 'unreadable') {
    return Promise.reject(new Error('fixture Cookies copy missing'));
  }
  if (process.env.ORDERCLI_LOAD_TEST_MODE === 'error') {
    return Promise.reject(new Error('fixture read failed'));
  }
  return Promise.resolve([{ name: 'fixture', value: '1' }]);
}

module.exports = { getCookiesPromised };
`

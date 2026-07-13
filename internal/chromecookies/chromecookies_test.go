package chromecookies

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadCookieHeader_Validation(t *testing.T) {
	_, err := LoadCookieHeader(context.Background(), Options{})
	if err == nil || !strings.Contains(err.Error(), "TargetURL missing") {
		t.Fatalf("expected target url missing, got %v", err)
	}

	_, err = LoadCookieHeader(context.Background(), Options{TargetURL: "https://example.invalid/"})
	if err == nil || !strings.Contains(err.Error(), "CacheDir missing") {
		t.Fatalf("expected cache dir missing, got %v", err)
	}
}

func TestEnsureNpmProject_ShortCircuit(t *testing.T) {
	dir := t.TempDir()
	seedNpmProject(t, dir)

	if err := ensureNpmProject(context.Background(), dir, nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestLoadCookieHeader_Success_WithStubRunner(t *testing.T) {
	orig := runScript
	defer func() { runScript = orig }()

	runScript = func(ctx context.Context, cacheDir, scriptPath, outPath string, input []byte, logWriter io.Writer, timeout time.Duration) (scriptOutput, error) {
		return scriptOutput{CookieHeader: "a=1; b=2", CookieCount: 2}, nil
	}

	cacheDir := t.TempDir()
	seedNpmProject(t, cacheDir)

	res, err := LoadCookieHeader(context.Background(), Options{
		TargetURL: "https://example.invalid/",
		CacheDir:  cacheDir,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.CookieCount != 2 || !strings.Contains(res.CookieHeader, "a=1") {
		t.Fatalf("unexpected: %#v", res)
	}
}

func TestLoadCookieHeader_StructuredError_WithStubRunner(t *testing.T) {
	orig := runScript
	defer func() { runScript = orig }()

	runScript = func(ctx context.Context, cacheDir, scriptPath, outPath string, input []byte, logWriter io.Writer, timeout time.Duration) (scriptOutput, error) {
		return scriptOutput{Error: "boom"}, nil
	}

	cacheDir := t.TempDir()
	seedNpmProject(t, cacheDir)

	_, err := LoadCookieHeader(context.Background(), Options{
		TargetURL: "https://example.invalid/",
		CacheDir:  cacheDir,
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestLoadCookieHeader_RealRunner_WithFakeNode(t *testing.T) {
	fakeBin := t.TempDir()
	writeExe(t, filepath.Join(fakeBin, "node"), `#!/bin/sh
set -e
cat > "$ORDERCLI_OUTPUT_PATH" <<'EOF'
{"cookie_header":"a=1; b=2","cookie_count":2}
EOF
exit 0
`)
	writeExe(t, filepath.Join(fakeBin, "npm"), "#!/bin/sh\nexit 0\n")

	withPATH(t, fakeBin)

	cacheDir := t.TempDir()
	seedNpmProject(t, cacheDir)

	res, err := LoadCookieHeader(context.Background(), Options{
		TargetURL: "https://example.invalid/",
		CacheDir:  cacheDir,
		Timeout:   500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.CookieCount != 2 || res.CookieHeader == "" {
		t.Fatalf("unexpected: %#v", res)
	}
}

func TestLoadCookieHeader_RealRunner_FakeNodeErrorWritesOut(t *testing.T) {
	fakeBin := t.TempDir()
	writeExe(t, filepath.Join(fakeBin, "node"), `#!/bin/sh
set -e
cat > "$ORDERCLI_OUTPUT_PATH" <<'EOF'
{"error":"no cookies"}
EOF
exit 2
`)
	writeExe(t, filepath.Join(fakeBin, "npm"), "#!/bin/sh\nexit 0\n")

	withPATH(t, fakeBin)

	cacheDir := t.TempDir()
	seedNpmProject(t, cacheDir)

	_, err := LoadCookieHeader(context.Background(), Options{
		TargetURL: "https://example.invalid/",
		CacheDir:  cacheDir,
		Timeout:   500 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "no cookies") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func withPATH(t *testing.T, prefixDir string) {
	t.Helper()
	old := os.Getenv("PATH")
	if err := os.Setenv("PATH", prefixDir+string(os.PathListSeparator)+old); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", old) })
}

func writeExe(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func seedNpmProject(t *testing.T, dir string) {
	t.Helper()
	p := filepath.Join(dir, "node_modules", "chrome-cookies-secure", "package.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write module package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(npmPackageJSON+"\n"), 0o600); err != nil {
		t.Fatalf("write root package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, npmStateFile), []byte(npmPackageJSON+"\n"), 0o600); err != nil {
		t.Fatalf("write dependency state: %v", err)
	}
}

func TestEnsureNpmProject_Installs_WithFakeNpm(t *testing.T) {
	fakeBin := t.TempDir()
	writeExe(t, filepath.Join(fakeBin, "npm"), `#!/bin/sh
set -e
mkdir -p node_modules/chrome-cookies-secure node_modules/tar node_modules/@tootallnate/once
echo '{"name":"chrome-cookies-secure","version":"3.0.2"}' > node_modules/chrome-cookies-secure/package.json
echo '{"name":"tar","version":"7.5.20"}' > node_modules/tar/package.json
echo '{"name":"@tootallnate/once","version":"2.0.1"}' > node_modules/@tootallnate/once/package.json
exit 0
`)
	withPATH(t, fakeBin)

	dir := t.TempDir()
	if err := ensureNpmProject(context.Background(), dir, nil); err != nil {
		t.Fatalf("ensureNpmProject: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		t.Fatalf("expected package.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules", "chrome-cookies-secure", "package.json")); err != nil {
		t.Fatalf("expected node_modules: %v", err)
	}
	pkg, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}
	for _, want := range []string{`"chrome-cookies-secure":"3.0.2"`, `"tar":"7.5.20"`, `"@tootallnate/once":"2.0.1"`} {
		if !strings.Contains(string(pkg), want) {
			t.Errorf("package.json missing %s: %s", want, pkg)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, npmStateFile)); err != nil {
		t.Fatalf("expected dependency state: %v", err)
	}
}

func TestEnsureNpmProject_RefreshesStaleConfig(t *testing.T) {
	fakeBin := t.TempDir()
	writeExe(t, filepath.Join(fakeBin, "npm"), `#!/bin/sh
set -e
touch npm-invoked
mkdir -p node_modules/chrome-cookies-secure node_modules/tar node_modules/@tootallnate/once
echo '{"name":"chrome-cookies-secure","version":"3.0.2"}' > node_modules/chrome-cookies-secure/package.json
echo '{"name":"tar","version":"7.5.20"}' > node_modules/tar/package.json
echo '{"name":"@tootallnate/once","version":"2.0.1"}' > node_modules/@tootallnate/once/package.json
exit 0
`)
	withPATH(t, fakeBin)

	dir := t.TempDir()
	p := filepath.Join(dir, "node_modules", "chrome-cookies-secure", "package.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write module package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"chrome-cookies-secure":"3.0.0"}}`), 0o600); err != nil {
		t.Fatalf("write stale package: %v", err)
	}

	if err := ensureNpmProject(context.Background(), dir, nil); err != nil {
		t.Fatalf("ensureNpmProject: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "npm-invoked")); err != nil {
		t.Fatalf("expected npm reinstall: %v", err)
	}
}

func TestVerifyInstalledNpmDependencies_RejectsStaleOverride(t *testing.T) {
	dir := t.TempDir()
	packages := map[string]string{
		"chrome-cookies-secure": `{"name":"chrome-cookies-secure","version":"3.0.2"}`,
		"tar":                   `{"name":"tar","version":"7.5.15"}`,
		"@tootallnate/once":     `{"name":"@tootallnate/once","version":"2.0.1"}`,
	}
	for name, metadata := range packages {
		p := filepath.Join(dir, "node_modules", filepath.FromSlash(name), "package.json")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(p, []byte(metadata), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	err := verifyInstalledNpmDependencies(dir)
	if err == nil || !strings.Contains(err.Error(), "installed tar version 7.5.15, want 7.5.20") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunScriptReal_NodeMissing(t *testing.T) {
	old := os.Getenv("PATH")
	_ = os.Setenv("PATH", "")
	t.Cleanup(func() { _ = os.Setenv("PATH", old) })

	_, err := runScriptReal(context.Background(), t.TempDir(), "x", "y", []byte("{}"), nil, 10*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "node not found") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestRunScriptReal_NpmMissing(t *testing.T) {
	fakeBin := t.TempDir()
	writeExe(t, filepath.Join(fakeBin, "node"), "#!/bin/sh\nexit 0\n")
	old := os.Getenv("PATH")
	_ = os.Setenv("PATH", fakeBin)
	t.Cleanup(func() { _ = os.Setenv("PATH", old) })

	_, err := runScriptReal(context.Background(), t.TempDir(), "x", "y", []byte("{}"), nil, 10*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "npm not found") {
		t.Fatalf("unexpected err: %v", err)
	}
}

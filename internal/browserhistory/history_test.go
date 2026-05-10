package browserhistory

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserLabel(t *testing.T) {
	if got := browserLabel(BrowserAtlas); got != "Atlas" {
		t.Fatalf("atlas=%q", got)
	}
	if got := browserLabel(BrowserChrome); got != "Chrome" {
		t.Fatalf("chrome=%q", got)
	}
	if got := browserLabel(BrowserAuto); got != "Atlas/Chrome" {
		t.Fatalf("auto=%q", got)
	}
}

func TestResolveLatestDeliverooStatusURL_UnsupportedBrowser(t *testing.T) {
	_, err := ResolveLatestDeliverooStatusURL(context.Background(), "safari")
	if err == nil || !strings.Contains(err.Error(), "unsupported browser") {
		t.Fatalf("err=%v", err)
	}
}

func TestLatestStatusURLFromHistory(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available")
	}

	dbPath := filepath.Join(t.TempDir(), "History")
	sql := `
create table urls (last_visit_time integer, url text);
insert into urls(last_visit_time, url) values
  (100, 'https://deliveroo.co.uk/orders/111/status'),
  (300, 'https://roo.it/s/abc'),
  (200, 'https://deliveroo.co.uk/orders/222/status');
`
	cmd := exec.Command("sqlite3", dbPath, sql) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sqlite init: %v out=%s", err, out)
	}

	got, err := latestStatusURLFromHistory(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("latestStatusURLFromHistory: %v", err)
	}
	if got.ts != 300 || got.url != "https://roo.it/s/abc" {
		t.Fatalf("got=%+v", got)
	}
}

func TestResolveLatestDeliverooStatusURL_AutoPrefersNewestAcrossBrowsers(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	atlasPath := filepath.Join(home, "Library", "Application Support", "com.openai.atlas.beta", "browser-data", "host", "user-1", "History")
	chromePath := filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default", "History")
	for _, p := range []string{atlasPath, chromePath} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}

	atlasSQL := `
create table urls (last_visit_time integer, url text);
insert into urls(last_visit_time, url) values (100, 'https://deliveroo.co.uk/orders/111/status');
`
	chromeSQL := `
create table urls (last_visit_time integer, url text);
insert into urls(last_visit_time, url) values (200, 'https://deliveroo.co.uk/orders/222/status');
`
	for path, sql := range map[string]string{atlasPath: atlasSQL, chromePath: chromeSQL} {
		cmd := exec.Command("sqlite3", path, sql) //nolint:gosec
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("sqlite init %s: %v out=%s", path, err, out)
		}
	}

	got, err := ResolveLatestDeliverooStatusURL(context.Background(), BrowserAuto)
	if err != nil {
		t.Fatalf("ResolveLatestDeliverooStatusURL: %v", err)
	}
	if got != "https://deliveroo.co.uk/orders/222/status" {
		t.Fatalf("got=%q", got)
	}
}

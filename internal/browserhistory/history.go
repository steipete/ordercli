package browserhistory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	BrowserAuto   = "auto"
	BrowserAtlas  = "atlas"
	BrowserChrome = "chrome"
)

type candidate struct {
	ts  int64
	url string
}

func ResolveLatestDeliverooStatusURL(ctx context.Context, browser string) (string, error) {
	browser = strings.ToLower(strings.TrimSpace(browser))
	if browser == "" {
		browser = BrowserAuto
	}

	var paths []string
	switch browser {
	case BrowserAuto:
		paths = append(paths, atlasHistoryPaths()...)
		paths = append(paths, chromeHistoryPaths()...)
	case BrowserAtlas:
		paths = atlasHistoryPaths()
	case BrowserChrome:
		paths = chromeHistoryPaths()
	default:
		return "", fmt.Errorf("unsupported browser %q", browser)
	}
	if len(paths) == 0 {
		return "", errors.New("no browser history files found")
	}

	var best candidate
	for _, path := range paths {
		c, err := latestStatusURLFromHistory(ctx, path)
		if err != nil {
			continue
		}
		if c.ts > best.ts {
			best = c
		}
	}
	if best.url == "" {
		return "", fmt.Errorf("no Deliveroo status URLs found in %s history", browserLabel(browser))
	}
	return best.url, nil
}

func browserLabel(browser string) string {
	switch browser {
	case BrowserAtlas:
		return "Atlas"
	case BrowserChrome:
		return "Chrome"
	default:
		return "Atlas/Chrome"
	}
}

func atlasHistoryPaths() []string {
	root := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "com.openai.atlas.beta", "browser-data", "host")
	matches, _ := filepath.Glob(filepath.Join(root, "*", "History"))
	return matches
}

func chromeHistoryPaths() []string {
	path := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Google", "Chrome", "Default", "History")
	if _, err := os.Stat(path); err == nil {
		return []string{path}
	}
	return nil
}

func latestStatusURLFromHistory(ctx context.Context, historyPath string) (candidate, error) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return candidate{}, errors.New("sqlite3 not found")
	}
	src, err := os.ReadFile(historyPath)
	if err != nil {
		return candidate{}, err
	}
	tmp, err := os.CreateTemp("", "ordercli-history-*.sqlite")
	if err != nil {
		return candidate{}, err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(src); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return candidate{}, err
	}
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	query := `select last_visit_time || '|' || url from urls where url like 'https://deliveroo.co.uk/orders/%/status%' or url like 'https://roo.it/s/%' order by last_visit_time desc limit 1;`
	cmd := exec.CommandContext(ctx, "sqlite3", "-batch", "-noheader", tmpPath, query) //nolint:gosec
	out, err := cmd.Output()
	if err != nil {
		return candidate{}, err
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return candidate{}, errors.New("no status url")
	}
	parts := strings.SplitN(line, "|", 2)
	if len(parts) != 2 {
		return candidate{}, fmt.Errorf("unexpected sqlite output %q", line)
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return candidate{}, err
	}
	return candidate{ts: ts, url: strings.TrimSpace(parts[1])}, nil
}

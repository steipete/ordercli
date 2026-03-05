package browserpage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadText_EmptyURL(t *testing.T) {
	_, err := ReadText(context.Background(), "   ", Options{})
	if err == nil || !strings.Contains(err.Error(), "url missing") {
		t.Fatalf("err=%v", err)
	}
}

func TestReadText_UsesDefaultsAndDecodesResult(t *testing.T) {
	orig := runFetchScriptFunc
	t.Cleanup(func() { runFetchScriptFunc = orig })

	runFetchScriptFunc = func(_ context.Context, _, _, _ string, input []byte, opts Options, playwright string) ([]byte, error) {
		var in scriptInput
		if err := json.Unmarshal(input, &in); err != nil {
			t.Fatalf("unmarshal input: %v", err)
		}
		if in.URL != "https://example.com" {
			t.Fatalf("url=%q", in.URL)
		}
		if in.TimeoutMillis != int((2 * time.Minute).Milliseconds()) {
			t.Fatalf("timeout=%d", in.TimeoutMillis)
		}
		if in.Headless {
			t.Fatalf("expected headless default false in input")
		}
		if opts.Timeout != 2*time.Minute {
			t.Fatalf("opts timeout=%s", opts.Timeout)
		}
		if playwright != "playwright@1.58.2" {
			t.Fatalf("playwright=%q", playwright)
		}
		return []byte(`{"final_url":"https://example.com/final","title":"T","text":"Body"}`), nil
	}

	got, err := ReadText(context.Background(), "https://example.com", Options{})
	if err != nil {
		t.Fatalf("ReadText: %v", err)
	}
	if got.FinalURL != "https://example.com/final" || got.Title != "T" || got.Text != "Body" {
		t.Fatalf("got=%+v", got)
	}
}

func TestReadText_InvalidJSON(t *testing.T) {
	orig := runFetchScriptFunc
	t.Cleanup(func() { runFetchScriptFunc = orig })

	runFetchScriptFunc = func(_ context.Context, _, _, _ string, _ []byte, _ Options, _ string) ([]byte, error) {
		return []byte(`{`), nil
	}

	_, err := ReadText(context.Background(), "https://example.com", Options{Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "decode output") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunFetchScript_NodeMissing(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := runFetchScript(context.Background(), t.TempDir(), "script.mjs", "out.json", nil, Options{Timeout: time.Second}, "playwright@1.58.2")
	if err == nil || !strings.Contains(err.Error(), "node not found") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunFetchScript_NpmMissing(t *testing.T) {
	binDir := t.TempDir()
	nodePath := filepath.Join(binDir, "node")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake node: %v", err)
	}
	t.Setenv("PATH", binDir)

	_, err := runFetchScript(context.Background(), t.TempDir(), "script.mjs", "out.json", nil, Options{Timeout: time.Second}, "playwright@1.58.2")
	if err == nil || !strings.Contains(err.Error(), "npm not found") {
		t.Fatalf("err=%v", err)
	}
}

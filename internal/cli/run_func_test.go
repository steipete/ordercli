package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steipete/ordercli/internal/version"
)

func TestRun_Help(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := Run(context.Background(), []string{"--config", cfgPath, "--help"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_VersionDoesNotLoadOrSaveConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	const invalidConfig = "not valid JSON\n"
	if err := os.WriteFile(cfgPath, []byte(invalidConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	out, errOut, err := runCLI(cfgPath, []string{"--version"}, "")
	if err != nil {
		t.Fatalf("version: %v stderr=%s", err, errOut)
	}
	if got, want := strings.TrimSpace(out), "ordercli version "+version.Version; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != invalidConfig {
		t.Fatalf("version changed config: %q", data)
	}
}

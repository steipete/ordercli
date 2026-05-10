package cli

import (
	"strings"
	"testing"
)

func TestFindPreset_CzechRepublic(t *testing.T) {
	p, ok := findPreset("CZ")
	if !ok {
		t.Fatal("expected CZ preset")
	}
	if p.BaseURL != "https://cz.fd-api.com/api/v5/" {
		t.Fatalf("BaseURL=%q", p.BaseURL)
	}
	if p.GlobalEntityID != "DJ_CZ" {
		t.Fatalf("GlobalEntityID=%q", p.GlobalEntityID)
	}
	if p.TargetISO != "CZ" {
		t.Fatalf("TargetISO=%q", p.TargetISO)
	}
}

func TestCountriesCmd_IncludesCzechRepublic(t *testing.T) {
	out, _, err := runCLI(t.TempDir()+"/config.json", []string{"foodora", "countries"}, "")
	if err != nil {
		t.Fatalf("countries: %v", err)
	}
	if !strings.Contains(out, "CZ\tDJ_CZ\thttps://cz.fd-api.com/api/v5/") {
		t.Fatalf("missing CZ preset in output:\n%s", out)
	}
}

func TestConfigSetCountry_CzechRepublic(t *testing.T) {
	cfgPath := t.TempDir() + "/config.json"
	if _, _, err := runCLI(cfgPath, []string{"foodora", "config", "set", "--country", "CZ"}, ""); err != nil {
		t.Fatalf("config set --country CZ: %v", err)
	}
	out, _, err := runCLI(cfgPath, []string{"foodora", "config", "show"}, "")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	for _, want := range []string{
		"base_url=https://cz.fd-api.com/api/v5/",
		"global_entity_id=DJ_CZ",
		"target_country_iso=CZ",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in config output:\n%s", want, out)
		}
	}
}

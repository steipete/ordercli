package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type countryPreset struct {
	Code           string
	BaseURL        string
	GlobalEntityID string
	TargetISO      string
}

var presets = []countryPreset{
	{Code: "HU", BaseURL: "https://hu.fd-api.com/api/v5/", GlobalEntityID: "NP_HU", TargetISO: "HU"},
	{Code: "SK", BaseURL: "https://sk.fd-api.com/api/v5/", GlobalEntityID: "FP_SK", TargetISO: "SK"},
	{Code: "DL", BaseURL: "https://dl.fd-api.com/api/v5/", GlobalEntityID: "FP_DE", TargetISO: "DE"},
	{Code: "AT", BaseURL: "https://mj.fd-api.com/api/v5/", GlobalEntityID: "MJM_AT", TargetISO: "AT"},
}

func newCountriesCmd(st *state) *cobra.Command {
	return &cobra.Command{
		Use:   "countries",
		Short: "List bundled country presets (from the APK)",
		Run: func(cmd *cobra.Command, args []string) {
			for _, p := range presets {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", p.Code, p.GlobalEntityID, p.BaseURL)
			}
		},
	}
}

func findPreset(code string) (countryPreset, bool) {
	for _, p := range presets {
		if p.Code == code {
			return p, true
		}
	}
	return countryPreset{}, false
}

package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newConfigCmd(st *state) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show/edit config",
	}
	cmd.AddCommand(newConfigShowCmd(st))
	cmd.AddCommand(newConfigSetCmd(st))
	return cmd
}

func newConfigShowCmd(st *state) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print current config (redacts tokens)",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := st.foodora()
			fmt.Fprintf(cmd.OutOrStdout(), "base_url=%s\n", cfg.BaseURL)
			fmt.Fprintf(cmd.OutOrStdout(), "global_entity_id=%s\n", cfg.GlobalEntityID)
			fmt.Fprintf(cmd.OutOrStdout(), "target_country_iso=%s\n", cfg.TargetCountryISO)
			fmt.Fprintf(cmd.OutOrStdout(), "device_id=%s\n", cfg.DeviceID)
			if cfg.AccessToken != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "access_token=***\n")
			}
			if cfg.RefreshToken != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "refresh_token=***\n")
			}
			if cfg.ClientSecret != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "client_secret=*** (stored)\n")
			}
			if cfg.OAuthClientID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "oauth_client_id=%s\n", cfg.OAuthClientID)
			}
			if cfg.HTTPUserAgent != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "http_user_agent=%s\n", cfg.HTTPUserAgent)
			}
			if len(cfg.CookiesByHost) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "cookies_by_host=*** (%d)\n", len(cfg.CookiesByHost))
			}
			if cfg.PendingMfaToken != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "pending_mfa=*** (%s, %s)\n", cfg.PendingMfaChannel, cfg.PendingMfaEmail)
			}
		},
	}
}

func newConfigSetCmd(st *state) *cobra.Command {
	var country string
	var baseURL string
	var globalEntityID string
	var targetISO string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update base URL / country",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := st.foodora()
			if country != "" {
				country = strings.ToUpper(country)
				p, ok := findPreset(country)
				if !ok {
					return fmt.Errorf("unknown country preset %q (see `ordercli foodora countries`)", country)
				}
				cfg.BaseURL = p.BaseURL
				cfg.GlobalEntityID = p.GlobalEntityID
				cfg.TargetCountryISO = p.TargetISO
				st.markDirty()
				return nil
			}

			if baseURL == "" && globalEntityID == "" && targetISO == "" {
				return errors.New("nothing to set (use --country or --base-url/--global-entity-id/--target-iso)")
			}
			if baseURL != "" {
				cfg.BaseURL = baseURL
			}
			if globalEntityID != "" {
				cfg.GlobalEntityID = globalEntityID
			}
			if targetISO != "" {
				cfg.TargetCountryISO = targetISO
			}
			st.markDirty()
			return nil
		},
	}

	cmd.Flags().StringVar(&country, "country", "", "country preset (HU, SK, DL, AT)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "API base URL (e.g. https://hu.fd-api.com/api/v5/)")
	cmd.Flags().StringVar(&globalEntityID, "global-entity-id", "", "X-Global-Entity-ID (e.g. NP_HU)")
	cmd.Flags().StringVar(&targetISO, "target-iso", "", "X-Target-Country-Code-ISO (e.g. HU)")
	return cmd
}

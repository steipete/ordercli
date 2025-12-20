package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newDeliverooConfigCmd(st *state) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show/edit Deliveroo config (non-secret)",
	}
	cmd.AddCommand(newDeliverooConfigShowCmd(st))
	cmd.AddCommand(newDeliverooConfigSetCmd(st))
	return cmd
}

func newDeliverooConfigShowCmd(st *state) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print current Deliveroo config",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := st.deliveroo()
			fmt.Fprintf(cmd.OutOrStdout(), "market=%s\n", cfg.Market)
			fmt.Fprintf(cmd.OutOrStdout(), "base_url=%s\n", cfg.BaseURL)
		},
	}
}

func newDeliverooConfigSetCmd(st *state) *cobra.Command {
	var market string
	var baseURL string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update market/base url",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := st.deliveroo()
			if strings.TrimSpace(market) == "" && strings.TrimSpace(baseURL) == "" {
				return fmt.Errorf("nothing to set (use --market and/or --base-url)")
			}
			if strings.TrimSpace(market) != "" {
				cfg.Market = strings.ToLower(strings.TrimSpace(market))
			}
			if strings.TrimSpace(baseURL) != "" {
				cfg.BaseURL = strings.TrimSpace(baseURL)
			}
			st.markDirty()
			return nil
		},
	}

	cmd.Flags().StringVar(&market, "market", "", "market (e.g. uk)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "API base url (default derived from market)")
	return cmd
}

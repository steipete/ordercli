package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/steipete/ordercli/internal/deliveroo"
)

func newDeliverooHistoryCmd(st *state) *cobra.Command {
	var market string
	var baseURL string
	var bearerToken string
	var cookie string
	var offset int
	var limit int
	var includeUgc bool
	var state string
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "history",
		Short: "List past orders (requires DELIVEROO_BEARER_TOKEN)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := st.deliveroo()

			m := strings.TrimSpace(market)
			if m == "" {
				m = strings.TrimSpace(cfg.Market)
			}
			b := strings.TrimSpace(bearerToken)
			if b == "" {
				b = strings.TrimSpace(os.Getenv("DELIVEROO_BEARER_TOKEN"))
			}
			if b == "" {
				return errors.New("missing bearer token (set DELIVEROO_BEARER_TOKEN or pass --bearer-token)")
			}
			c := strings.TrimSpace(cookie)
			if c == "" {
				c = strings.TrimSpace(os.Getenv("DELIVEROO_COOKIE"))
			}

			u := strings.TrimSpace(baseURL)
			if u == "" {
				u = strings.TrimSpace(cfg.BaseURL)
			}

			cl, err := deliveroo.NewClient(deliveroo.ClientOptions{
				BaseURL:     u,
				Market:      m,
				BearerToken: b,
				Cookie:      c,
				Timeout:     20 * time.Second,
			})
			if err != nil {
				return err
			}

			resp, err := cl.OrderHistory(cmd.Context(), deliveroo.OrderHistoryParams{
				Offset:     offset,
				Limit:      limit,
				IncludeUgc: includeUgc,
				State:      strings.TrimSpace(state),
			})
			if err != nil {
				return err
			}

			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			out := cmd.OutOrStdout()
			if len(resp.Orders) == 0 {
				fmt.Fprintln(out, "no orders")
				return nil
			}
			for _, o := range resp.Orders {
				fmt.Fprintln(out, o.Summary())
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&market, "market", "", "market (default: config market)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "API base url (default: config base_url or derived from market)")
	cmd.Flags().StringVar(&bearerToken, "bearer-token", "", "bearer token (or set DELIVEROO_BEARER_TOKEN)")
	cmd.Flags().StringVar(&cookie, "cookie", "", "cookie header (or set DELIVEROO_COOKIE)")
	cmd.Flags().IntVar(&offset, "offset", 0, "paging offset")
	cmd.Flags().IntVar(&limit, "limit", 10, "paging limit")
	cmd.Flags().BoolVar(&includeUgc, "include-ugc", false, "include UGC in response")
	cmd.Flags().StringVar(&state, "state", "", "state filter (provider-specific)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print raw JSON")
	return cmd
}

func newDeliverooOrdersCmd(st *state) *cobra.Command {
	var interval time.Duration
	var once bool
	var browser string
	var statusURL string
	var asJSON bool

	cmd := &cobra.Command{
		Use:     "orders",
		Aliases: []string{"active"},
		Short:   "List active orders",
		RunE: func(cmd *cobra.Command, args []string) error {
			runOnce := func() error {
				if strings.TrimSpace(os.Getenv("DELIVEROO_BEARER_TOKEN")) != "" {
					h := newDeliverooHistoryCmd(st)
					h.SetArgs([]string{"--state", "active"})
					h.SetOut(cmd.OutOrStdout())
					h.SetErr(cmd.ErrOrStderr())
					h.SetContext(cmd.Context())
					return h.Execute()
				}

				targetURL := strings.TrimSpace(statusURL)
				if targetURL == "" {
					u, err := deliverooResolveLatestStatusURL(cmd.Context(), browser)
					if err != nil {
						return err
					}
					targetURL = u
				}

				status, err := deliverooFetchPublicStatus(cmd.Context(), targetURL, 2*time.Minute)
				if err != nil {
					return err
				}

				if asJSON {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					return enc.Encode(status)
				}

				fmt.Fprintln(cmd.OutOrStdout(), status.DetailsString())
				return nil
			}

			if interval <= 0 || once {
				return runOnce()
			}

			for {
				if err := runOnce(); err != nil {
					return err
				}
				time.Sleep(interval)
			}
		},
	}

	cmd.Flags().DurationVar(&interval, "interval", 0, "poll interval (default: once)")
	cmd.Flags().BoolVar(&once, "once", false, "fetch once (default)")
	cmd.Flags().StringVar(&browser, "browser", "auto", "browser history to inspect when no bearer token is set (auto, atlas, chrome)")
	cmd.Flags().StringVar(&statusURL, "status-url", "", "explicit Deliveroo status/share URL")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print raw JSON")
	return cmd
}

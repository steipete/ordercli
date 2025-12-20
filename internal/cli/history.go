package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/steipete/ordercli/internal/foodora"
)

func newHistoryCmd(st *state) *cobra.Command {
	var totalLimit int
	var pageSize int
	var include string
	var pandagoEnabled bool

	cmd := &cobra.Command{
		Use:   "history",
		Short: "List past orders",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(st)
			if err != nil {
				return err
			}

			limit := totalLimit
			if limit <= 0 {
				limit = 20
			}
			ps := pageSize
			if ps <= 0 {
				ps = 20
			}
			if ps > 100 {
				ps = 100
			}

			out := cmd.OutOrStdout()
			ctx := cmd.Context()

			offset := 0
			printed := 0
			for printed < limit {
				reqLimit := min(ps, limit-printed)
				resp, err := c.OrderHistory(ctx, foodora.OrderHistoryRequest{
					Include:        include,
					Offset:         offset,
					Limit:          reqLimit,
					PandaGoEnabled: pandagoEnabled,
				})
				if err != nil {
					return err
				}

				if len(resp.Data.Items) == 0 {
					if printed == 0 {
						fmt.Fprintln(out, "no past orders")
					}
					return nil
				}

				for _, o := range resp.Data.Items {
					if printed >= limit {
						return nil
					}
					fmt.Fprintf(out, "%s\t%s\t%s\t%s\n",
						o.OrderCode,
						historyVendor(o.Vendor),
						historyStatus(o.CurrentStatus),
						historyTime(o.ConfirmedDeliveryTime),
					)
					printed++
				}

				offset += len(resp.Data.Items)
				if resp.Data.TotalCount > 0 && offset >= int(resp.Data.TotalCount) {
					return nil
				}
				if len(resp.Data.Items) < reqLimit {
					return nil
				}
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&totalLimit, "limit", 20, "max orders to print")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "page size (API limit)")
	cmd.Flags().StringVar(&include, "include", "order_products,order_details", "include fields")
	cmd.Flags().BoolVar(&pandagoEnabled, "pandago-enabled", false, "set pandago_enabled=true")

	cmd.AddCommand(newHistoryShowCmd(st))
	return cmd
}

func historyVendor(v *foodora.OrderHistoryVendor) string {
	if v == nil {
		return ""
	}
	return v.Name
}

func historyStatus(s *foodora.OrderHistoryStatus) string {
	if s == nil {
		return ""
	}
	if s.Message != "" {
		return s.Message
	}
	if s.Code != "" {
		return string(s.Code)
	}
	return string(s.InternalStatusCode)
}

func historyTime(t *foodora.OrderHistoryTime) string {
	if t == nil || t.Date.Time.IsZero() {
		return ""
	}
	return t.Date.Time.In(time.Local).Format(time.RFC3339)
}

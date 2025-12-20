package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steipete/ordercli/internal/foodora"
)

func newHistoryShowCmd(st *state) *cobra.Command {
	var include string
	var itemReplacement bool
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "show <orderCode>",
		Short: "Show details for a historical order (orders/order_history?order_code=...)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(st)
			if err != nil {
				return err
			}

			resp, err := c.OrderHistoryByCode(cmd.Context(), foodora.OrderHistoryByCodeRequest{
				OrderCode:       args[0],
				Include:         include,
				ItemReplacement: itemReplacement,
			})
			if err != nil {
				return err
			}
			if len(resp.Data.Items) == 0 {
				return errors.New("no order found")
			}

			item := resp.Data.Items[0]
			out := cmd.OutOrStdout()

			if asJSON {
				b, _ := json.MarshalIndent(item, "", "  ")
				b = append(b, '\n')
				_, _ = out.Write(b)
				return nil
			}

			printHistoryDetail(out, item)
			return nil
		},
	}

	cmd.Flags().StringVar(&include, "include", "order_products,order_details", "include fields")
	cmd.Flags().BoolVar(&itemReplacement, "item-replacement", false, "set item_replacement=true")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print raw JSON")
	return cmd
}

func printHistoryDetail(out io.Writer, item map[string]any) {
	code := asString(item["order_code"])
	vendor := asString(nested(item, "vendor", "name"))
	status := asString(nested(item, "current_status", "message"))
	if status == "" {
		status = asString(nested(item, "current_status", "code"))
	}
	when := formatOrderTime(nested(item, "confirmed_delivery_time", "date"))
	total := formatMoney(item["total_value"])
	if total == "" {
		total = formatMoney(nested(item, "payment", "total_value"))
	}

	fmt.Fprintf(out, "order=%s\n", code)
	if vendor != "" {
		fmt.Fprintf(out, "vendor=%s\n", vendor)
	}
	if when != "" {
		fmt.Fprintf(out, "time=%s\n", when)
	}
	if status != "" {
		fmt.Fprintf(out, "status=%s\n", status)
	}
	if total != "" {
		fmt.Fprintf(out, "total=%s\n", total)
	}

	products, _ := item["order_products"].([]any)
	if len(products) > 0 {
		fmt.Fprintln(out, "items:")
		for _, raw := range products {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			name := asString(m["name"])
			if name == "" {
				name = asString(m["title"])
			}
			qty := asInt(m["quantity"])
			line := formatMoney(m["total_price"])
			if line == "" {
				line = formatMoney(m["total_value"])
			}
			if line == "" {
				line = formatMoney(m["price"])
			}

			switch {
			case qty > 0 && line != "" && name != "":
				fmt.Fprintf(out, "- %dx %s (%s)\n", qty, name, line)
			case qty > 0 && name != "":
				fmt.Fprintf(out, "- %dx %s\n", qty, name)
			case name != "":
				fmt.Fprintf(out, "- %s\n", name)
			}
		}
	}

	if addr := asString(item["order_address"]); addr != "" {
		fmt.Fprintf(out, "address=%s\n", addr)
	}

	// Helpful: show keys when we can't parse much.
	if vendor == "" && len(products) == 0 {
		keys := make([]string, 0, len(item))
		for k := range item {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(out, "keys=%s\n", strings.Join(keys, ","))
	}
}

func asString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(t, 'f', -1, 64))
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func asInt(v any) int {
	switch t := v.(type) {
	case nil:
		return 0
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(t))
		return i
	default:
		return 0
	}
}

func nested(m map[string]any, keys ...string) any {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	return cur
}

func formatOrderTime(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return ""
		}
		if tt, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return tt.In(time.Local).Format(time.RFC3339)
		}
		if tt, err := time.Parse(time.RFC3339, s); err == nil {
			return tt.In(time.Local).Format(time.RFC3339)
		}
		return s
	case float64:
		iv := int64(t)
		if iv > 1_000_000_000_000 {
			return time.UnixMilli(iv).In(time.Local).Format(time.RFC3339)
		}
		return time.Unix(iv, 0).In(time.Local).Format(time.RFC3339)
	default:
		return asString(v)
	}
}

func formatMoney(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case float64:
		if t == 0 {
			return ""
		}
		return strconv.FormatFloat(t, 'f', 2, 64)
	case int:
		if t == 0 {
			return ""
		}
		return strconv.FormatFloat(float64(t), 'f', 2, 64)
	case string:
		s := strings.TrimSpace(t)
		if s == "" || s == "0" || s == "0.0" || s == "0.00" {
			return ""
		}
		return s
	default:
		return ""
	}
}

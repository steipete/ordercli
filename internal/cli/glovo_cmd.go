package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/steipete/ordercli/internal/glovo"
)

func newGlovoCmd(st *state) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "glovo",
		Short: "Glovo",
	}
	cmd.AddCommand(newGlovoConfigCmd(st))
	cmd.AddCommand(newGlovoSessionCmd(st))
	cmd.AddCommand(newGlovoLogoutCmd(st))
	cmd.AddCommand(newGlovoHistoryCmd(st))
	cmd.AddCommand(newGlovoOrderCmd(st))
	cmd.AddCommand(newGlovoOrdersCmd(st))
	cmd.AddCommand(newGlovoCartCmd(st))
	cmd.AddCommand(newGlovoMeCmd(st))
	return cmd
}

func newGlovoClient(st *state) (*glovo.Client, error) {
	cfg := st.glovo()
	return glovo.New(glovo.Options{
		BaseURL:     cfg.BaseURL,
		AccessToken: cfg.AccessToken,
		DeviceURN:   cfg.DeviceURN,
		CityCode:    cfg.CityCode,
		CountryCode: cfg.CountryCode,
		Language:    cfg.Language,
		Latitude:    cfg.Latitude,
		Longitude:   cfg.Longitude,
	})
}

// Config commands

func newGlovoConfigCmd(st *state) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show/edit Glovo config",
	}
	cmd.AddCommand(newGlovoConfigShowCmd(st))
	cmd.AddCommand(newGlovoConfigSetCmd(st))
	return cmd
}

func newGlovoConfigShowCmd(st *state) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print current Glovo config",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := st.glovo()
			fmt.Fprintf(cmd.OutOrStdout(), "base_url=%s\n", cfg.BaseURL)
			fmt.Fprintf(cmd.OutOrStdout(), "city_code=%s\n", cfg.CityCode)
			fmt.Fprintf(cmd.OutOrStdout(), "country_code=%s\n", cfg.CountryCode)
			fmt.Fprintf(cmd.OutOrStdout(), "language=%s\n", cfg.Language)
			fmt.Fprintf(cmd.OutOrStdout(), "latitude=%v\n", cfg.Latitude)
			fmt.Fprintf(cmd.OutOrStdout(), "longitude=%v\n", cfg.Longitude)
			if cfg.AccessToken != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "access_token=%s...\n", cfg.AccessToken[:min(20, len(cfg.AccessToken))])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "access_token=(not set)\n")
			}
		},
	}
}

func newGlovoConfigSetCmd(st *state) *cobra.Command {
	var cityCode, countryCode, language, baseURL string
	var lat, lon float64

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update Glovo config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := st.glovo()
			changed := false

			if strings.TrimSpace(cityCode) != "" {
				cfg.CityCode = strings.ToUpper(strings.TrimSpace(cityCode))
				changed = true
			}
			if strings.TrimSpace(countryCode) != "" {
				cfg.CountryCode = strings.ToUpper(strings.TrimSpace(countryCode))
				changed = true
			}
			if strings.TrimSpace(language) != "" {
				cfg.Language = strings.ToLower(strings.TrimSpace(language))
				changed = true
			}
			if strings.TrimSpace(baseURL) != "" {
				cfg.BaseURL = strings.TrimSpace(baseURL)
				changed = true
			}
			if cmd.Flags().Changed("lat") {
				cfg.Latitude = lat
				changed = true
			}
			if cmd.Flags().Changed("lon") {
				cfg.Longitude = lon
				changed = true
			}

			if !changed {
				return fmt.Errorf("nothing to set (use --city-code, --country-code, --language, --lat, --lon, or --base-url)")
			}
			st.markDirty()
			fmt.Fprintln(cmd.OutOrStdout(), "config updated")
			return nil
		},
	}

	cmd.Flags().StringVar(&cityCode, "city-code", "", "city code (e.g. MAD)")
	cmd.Flags().StringVar(&countryCode, "country-code", "", "country code (e.g. ES)")
	cmd.Flags().StringVar(&language, "language", "", "language code (e.g. en)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "API base URL")
	cmd.Flags().Float64Var(&lat, "lat", 0, "delivery latitude")
	cmd.Flags().Float64Var(&lon, "lon", 0, "delivery longitude")
	return cmd
}

// Session command

func newGlovoSessionCmd(st *state) *cobra.Command {
	return &cobra.Command{
		Use:   "session <access_token>",
		Short: "Set access token (from browser localStorage glovo_auth_info)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token := strings.TrimSpace(args[0])
			if token == "" {
				return fmt.Errorf("access token cannot be empty")
			}

			cfg := st.glovo()
			cfg.AccessToken = token
			st.markDirty()

			fmt.Fprintln(cmd.OutOrStdout(), "access token saved")
			return nil
		},
	}
}

// Logout command

func newGlovoLogoutCmd(st *state) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear stored access token",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := st.glovo()
			cfg.AccessToken = ""
			cfg.DeviceURN = ""
			st.markDirty()
			fmt.Fprintln(cmd.OutOrStdout(), "logged out")
		},
	}
}

// History command

func newGlovoHistoryCmd(st *state) *cobra.Command {
	var offset, limit int
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "history",
		Short: "List past orders",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := newGlovoClient(st)
			if err != nil {
				return err
			}

			resp, err := cl.OrderHistory(cmd.Context(), offset, limit)
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
				title := o.Content.Title
				price := ""
				if o.Footer.Left != nil {
					price = o.Footer.Left.DataString()
				}

				items := ""
				if len(o.Content.Body) > 0 {
					// Get first few items
					itemText := o.Content.Body[0].Data
					lines := strings.Split(itemText, "\n")
					if len(lines) > 3 {
						items = strings.Join(lines[:3], ", ") + "..."
					} else {
						items = strings.Join(lines, ", ")
					}
				}

				fmt.Fprintf(out, "[%d] %s - %s\n", o.OrderID, title, price)
				if items != "" {
					fmt.Fprintf(out, "    %s\n", items)
				}
				fmt.Fprintln(out)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&offset, "offset", 0, "paging offset")
	cmd.Flags().IntVar(&limit, "limit", 12, "paging limit")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print raw JSON")
	return cmd
}

// Order command (single order details)

func newGlovoOrderCmd(st *state) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "order <orderID>",
		Short: "Show details for a single order",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var orderID int
			if _, err := fmt.Sscanf(args[0], "%d", &orderID); err != nil {
				return fmt.Errorf("invalid order ID: %s", args[0])
			}

			cl, err := newGlovoClient(st)
			if err != nil {
				return err
			}

			order, err := cl.GetOrder(cmd.Context(), orderID)
			if err != nil {
				return err
			}

			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(order)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Order ID: %d\n", order.OrderID)
			fmt.Fprintf(out, "Store: %s\n", order.Content.Title)
			fmt.Fprintf(out, "Status: %s\n", order.LayoutType)
			if order.Footer.Left != nil {
				fmt.Fprintf(out, "Total: %s\n", order.Footer.Left.DataString())
			}
			if order.CourierName != nil {
				fmt.Fprintf(out, "Courier: %s\n", *order.CourierName)
			}

			if len(order.Content.Body) > 0 {
				fmt.Fprintln(out, "\nItems:")
				for _, b := range order.Content.Body {
					lines := strings.Split(b.Data, "\n")
					for _, line := range lines {
						if line = strings.TrimSpace(line); line != "" {
							fmt.Fprintf(out, "  - %s\n", line)
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print raw JSON")
	return cmd
}

// Orders command (active orders tracking)

func newGlovoOrdersCmd(st *state) *cobra.Command {
	var asJSON bool
	var watch bool
	var interval int

	cmd := &cobra.Command{
		Use:   "orders",
		Short: "Show active orders (being delivered)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := newGlovoClient(st)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()

			printOrders := func() error {
				orders, err := cl.ActiveOrders(cmd.Context())
				if err != nil {
					return err
				}

				if asJSON {
					enc := json.NewEncoder(out)
					enc.SetIndent("", "  ")
					return enc.Encode(orders)
				}

				if len(orders) == 0 {
					fmt.Fprintln(out, "no active orders")
					return nil
				}

				for _, o := range orders {
					title := o.Content.Title
					status := o.LayoutType

					fmt.Fprintf(out, "[%d] %s\n", o.OrderID, title)
					fmt.Fprintf(out, "    Status: %s\n", status)
					if o.CourierName != nil {
						fmt.Fprintf(out, "    Courier: %s\n", *o.CourierName)
					}
					fmt.Fprintln(out)
				}
				return nil
			}

			if watch {
				for {
					// Clear screen for fresh output
					fmt.Fprint(out, "\033[2J\033[H")
					fmt.Fprintf(out, "Active Orders (refreshing every %ds, Ctrl+C to stop)\n\n", interval)
					if err := printOrders(); err != nil {
						fmt.Fprintf(out, "Error: %v\n", err)
					}
					select {
					case <-cmd.Context().Done():
						return nil
					case <-time.After(time.Duration(interval) * time.Second):
						// Continue to next iteration
					}
				}
			}

			return printOrders()
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print raw JSON")
	cmd.Flags().BoolVar(&watch, "watch", false, "continuously poll for updates")
	cmd.Flags().IntVar(&interval, "interval", 30, "polling interval in seconds (with --watch)")
	return cmd
}

// Cart command

func newGlovoCartCmd(st *state) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "cart",
		Short: "Show shopping cart (saved baskets)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := newGlovoClient(st)
			if err != nil {
				return err
			}

			// First get user ID
			user, err := cl.Me(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			baskets, err := cl.Baskets(cmd.Context(), user.ID)
			if err != nil {
				return err
			}

			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(baskets)
			}

			out := cmd.OutOrStdout()
			if len(baskets) == 0 {
				fmt.Fprintln(out, "cart is empty")
				return nil
			}

			for _, b := range baskets {
				fmt.Fprintf(out, "Store: %s (ID: %d)\n", b.StoreName, b.StoreID)
				fmt.Fprintf(out, "  Items:\n")
				for _, p := range b.Products {
					fmt.Fprintf(out, "    %dx %s - %.2f %s\n", p.Quantity, p.Name, p.TotalPrice, b.Currency)
				}
				fmt.Fprintf(out, "  Subtotal: %.2f %s\n", b.SubTotal, b.Currency)
				if b.DeliveryFee > 0 {
					fmt.Fprintf(out, "  Delivery: %.2f %s\n", b.DeliveryFee, b.Currency)
				}
				if b.ServiceFee > 0 {
					fmt.Fprintf(out, "  Service: %.2f %s\n", b.ServiceFee, b.Currency)
				}
				fmt.Fprintf(out, "  Total: %.2f %s\n", b.Total, b.Currency)
				if !b.IsMinOrderMet {
					fmt.Fprintf(out, "  ! Min order: %.2f %s\n", b.MinOrderValue, b.Currency)
				}
				fmt.Fprintln(out)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print raw JSON")
	return cmd
}

// Me command

func newGlovoMeCmd(st *state) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show current user profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := newGlovoClient(st)
			if err != nil {
				return err
			}

			user, err := cl.Me(cmd.Context())
			if err != nil {
				return err
			}

			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(user)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "ID: %d\n", user.ID)
			fmt.Fprintf(out, "Name: %s\n", user.Name)
			fmt.Fprintf(out, "Email: %s\n", user.Email)
			if user.PhoneNumber != nil {
				fmt.Fprintf(out, "Phone: %s\n", user.PhoneNumber.Number)
			}
			fmt.Fprintf(out, "City: %s\n", user.PreferredCityCode)
			fmt.Fprintf(out, "Language: %s\n", user.PreferredLanguage)
			fmt.Fprintf(out, "Orders: %d\n", user.DeliveredOrdersCount)

			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print raw JSON")
	return cmd
}

package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steipete/ordercli/internal/config"
	"github.com/steipete/ordercli/internal/foodora"
	"github.com/steipete/ordercli/internal/version"
)

func newOrdersCmd(st *state) *cobra.Command {
	var watch bool

	cmd := &cobra.Command{
		Use:   "orders",
		Short: "List active orders",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(st)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			for {
				resp, err := c.ActiveOrders(ctx)
				if err != nil {
					return err
				}
				printActiveOrders(cmd, resp.Data.ActiveOrders)

				if !watch {
					return nil
				}
				sleep := 30 * time.Second
				if resp.Data.PollInSeconds != nil && *resp.Data.PollInSeconds > 0 {
					sleep = time.Duration(*resp.Data.PollInSeconds) * time.Second
				}
				time.Sleep(sleep)
			}
		},
	}
	cmd.Flags().BoolVar(&watch, "watch", false, "poll active orders")
	return cmd
}

func newOrderCmd(st *state) *cobra.Command {
	return &cobra.Command{
		Use:   "order <orderCode>",
		Short: "Show details for a single order (tracking/orders/{orderCode})",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(st)
			if err != nil {
				return err
			}
			resp, err := c.OrderStatus(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "status=%d\n", resp.Status)
			if v, ok := resp.Data["status_messages"]; ok {
				fmt.Fprintf(cmd.OutOrStdout(), "status_messages=%v\n", v)
			}
			return nil
		},
	}
}

func newAuthedClient(st *state) (*foodora.Client, error) {
	cfg := st.foodora()
	if cfg.BaseURL == "" {
		return nil, errors.New("missing base_url (run `ordercli foodora config set --country ...`)")
	}
	if !cfg.HasSession() {
		return nil, errors.New("not logged in (run `ordercli foodora login ...`)")
	}

	_, cookie := st.cookieHeaderForBaseURL()
	prof := st.appHeaders()
	ua := cfg.HTTPUserAgent
	if ua == "" && prof.UserAgent != "" {
		ua = prof.UserAgent
	}
	if ua == "" {
		ua = "ordercli/" + version.Version
	}

	c, err := foodora.New(foodora.Options{
		BaseURL:          cfg.BaseURL,
		DeviceID:         cfg.DeviceID,
		GlobalEntityID:   cfg.GlobalEntityID,
		TargetCountryISO: cfg.TargetCountryISO,
		AccessToken:      cfg.AccessToken,
		UserAgent:        ua,
		CookieHeader:     cookie,
		FPAPIKey:         prof.FPAPIKey,
		AppName:          prof.AppName,
		OriginalUserAgent: func() string {
			if strings.HasPrefix(ua, "Android-app-") {
				return ua
			}
			return ""
		}(),
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if cfg.TokenLikelyExpired(now) {
		sec, err := st.resolveClientSecret(context.Background(), cfg.OAuthClientID)
		if err != nil {
			return nil, err
		}
		tok, err := c.OAuthTokenRefresh(context.Background(), foodora.OAuthRefreshRequest{
			RefreshToken: cfg.RefreshToken,
			ClientSecret: sec.Secret,
			ClientID:     cfg.OAuthClientID,
		})
		if err != nil && isInvalidClientErr(err) {
			if sec2, ferr := st.forceFetchClientSecret(context.Background(), cfg.OAuthClientID); ferr == nil {
				tok, err = c.OAuthTokenRefresh(context.Background(), foodora.OAuthRefreshRequest{
					RefreshToken: cfg.RefreshToken,
					ClientSecret: sec2.Secret,
					ClientID:     cfg.OAuthClientID,
				})
			}
		}
		if err != nil {
			return nil, err
		}
		cfg.AccessToken = tok.AccessToken
		cfg.RefreshToken = tok.RefreshToken
		cfg.ExpiresAt = tok.ExpiresAt(now)
		if cfg.ExpiresAt.IsZero() {
			if exp, ok := config.AccessTokenExpiresAt(tok.AccessToken); ok {
				cfg.ExpiresAt = exp
			}
		}
		st.markDirty()
		c.SetAccessToken(tok.AccessToken)
	}

	return c, nil
}

func printActiveOrders(cmd *cobra.Command, orders []foodora.ActiveOrder) {
	out := cmd.OutOrStdout()
	if len(orders) == 0 {
		fmt.Fprintln(out, "no active orders")
		return
	}
	for _, o := range orders {
		status := o.Status.Subtitle
		if status == "" && len(o.Status.Titles) > 0 {
			status = o.Status.Titles[0].Name
		}
		fmt.Fprintf(out, "%s\t%s\t%s\n", o.Code, o.Vendor.Name, status)
	}
}

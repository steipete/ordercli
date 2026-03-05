package cli

import (
	"context"
	"time"

	"github.com/steipete/ordercli/internal/browserauth"
	"github.com/steipete/ordercli/internal/browserhistory"
	"github.com/steipete/ordercli/internal/chromecookies"
	"github.com/steipete/ordercli/internal/deliveroo"
	"github.com/steipete/ordercli/internal/foodora"
)

var chromeLoadCookieHeader = chromecookies.LoadCookieHeader

var browserOAuthTokenPassword = func(ctx context.Context, req foodora.OAuthPasswordRequest, opts browserauth.PasswordOptions) (foodora.AuthToken, *foodora.MfaChallenge, browserauth.Session, error) {
	return browserauth.OAuthTokenPassword(ctx, req, opts)
}

var deliverooResolveLatestStatusURL = browserhistory.ResolveLatestDeliverooStatusURL

var deliverooFetchPublicStatus = func(ctx context.Context, targetURL string, timeout time.Duration) (deliveroo.PublicStatus, error) {
	return deliveroo.FetchPublicStatus(ctx, targetURL, timeout)
}

# Changelog

## Unreleased

- Release binaries for macOS are now Developer ID signed and notarized, so direct downloads pass Gatekeeper.

## 0.2.0 (2026-09-13)

**Highlights:** Foodora gains Sweden and Czech Republic presets, Chrome cookie imports work with npm 12 native builds, and macOS binaries now require macOS 13 or newer.

- Add Sweden (`SE`) Foodora preset using `OP_SE`. (`#4`, thanks `@grenish`)
- Add Czech Republic (`CZ`) Foodora preset using `DJ_CZ`. (`#6`, thanks `@usimic`)
- CLI: add `--version` without loading or saving user configuration.
- Chrome cookies: explicitly approve native dependency builds for npm 12 and rebuild cached installs that skipped them.
- macOS: prebuilt binaries now require macOS 13 or newer with the Go 1.27 toolchain.
- Build: prefer Go 1.27.1 while retaining Go 1.27.0 source support, test both versions in CI, and update the Dockerfile frontend to 1.27.
- Dependencies: update Go to 1.27, Docker Node to 26.8.1, npm to 12.0.2, Playwright to 1.62.1, current Go terminal/system libraries, and cached Chrome cookie support to 3.0.2 with audited transitive overrides (tar 7.5.22).
- Docker: add a local image with Node, Playwright Chromium, `/data` persistence, and CI smoke coverage.
- Release: update GoReleaser archive configuration and document the automated Homebrew tap handoff.

## 0.1.0 (2025-12-20)

- Initial CLI (`login`, `orders`, `order`, `config`, `countries`)
- Rename project to `ordercli` + provider-first commands (`ordercli <provider> ...`)
- Past orders (`history` via `orders/order_history`)
- Historical order details (`history show <orderCode>`)
- Auto-fetch/cache OAuth `client_secret` from Firebase Remote Config
- OAuth token flow with refresh + MFA detection (`mfa_triggered`)
- Interactive OTP prompt + retry (TTY)
- Order tracking endpoints (`tracking/active-orders`, `tracking/orders/{orderCode}`)
- Optional Playwright interactive login (`--browser`) + Cloudflare cookie capture (e.g. Austria/mjam)
- Persistent Playwright profile support (`--browser-profile`)
- `--config` flag works (use separate config files for testing)
- OAuth `--client-id` override (e.g. `corp_android`)
- Reorder: preview by default; `orders/{orderCode}/reorder` with `--confirm` (adds to cart; address selectable)
- Deliveroo (basic/WIP): `deliveroo history` (requires `DELIVEROO_BEARER_TOKEN`)
- Deliveroo: accept numeric `order_number` values returned by the UK order history API.
- Glovo (basic/WIP): config, token session, history, active orders, order details, cart, and profile commands
- Tests: reorder + redaction regressions

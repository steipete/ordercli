# Changelog

## Unreleased

- Dependencies: update Go to 1.26, Playwright to 1.61.1, current Go terminal/system libraries, and cached Chrome cookie support to 3.0.2 with audited transitive overrides.
- Docker: add a local image with Node, Playwright Chromium, `/data` persistence, and CI smoke coverage.
- Add Sweden (`SE`) Foodora preset using `OP_SE`. (`#4`, thanks `@grenish`)
- Add Czech Republic (`CZ`) Foodora preset using `DJ_CZ`. (`#6`, thanks `@usimic`)

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

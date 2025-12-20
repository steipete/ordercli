# ordercli 🛵

Your takeout timeline, in the terminal.

Repo: `github.com/steipete/ordercli`

Providers:
- `foodora` (working)
- `deliveroo` (work in progress; requires `DELIVEROO_BEARER_TOKEN`)

Concepts (shared CLI UX; provider-specific implementations):
- `history` (past orders)
- `orders` (active orders)
- `order` / `history show` (details)

## Build

```sh
go test ./...
go build ./cmd/ordercli
```

## foodora

### Configure country / base URL

Bundled presets (from the APK):

```sh
./ordercli foodora countries
./ordercli foodora config set --country HU
./ordercli foodora config set --country AT
./ordercli foodora config show
```

Manual:

```sh
./ordercli foodora config set --base-url https://hu.fd-api.com/api/v5/ --global-entity-id NP_HU --target-iso HU
```

### Login

`oauth2/token` needs a `client_secret` (the app fetches it via remote config). `ordercli` auto-fetches it on first use and caches it locally.

Optional override (keeps secrets out of shell history):

```sh
export FOODORA_CLIENT_SECRET='...'
./ordercli foodora login --email you@example.com --password-stdin
```

If MFA triggers and you're running in a TTY, `ordercli` prompts for the OTP code and retries automatically. Otherwise it stores the MFA token locally and prints a safe retry command (`--otp <CODE>`).

### Client headers

Some regions (e.g. Austria/mjam `mj.fd-api.com`) expect app-style headers like `X-FP-API-KEY` / `App-Name` / app `User-Agent`. `ordercli` uses an app-like header profile for `AT` by default.

For corporate flows, you can override the OAuth `client_id`:

```sh
./ordercli foodora login --email you@example.com --client-id corp_android --password-stdin
```

### Cloudflare / bot protection

Some regions (e.g. Austria/mjam `mj.fd-api.com`) may return Cloudflare HTML (`HTTP 403`) for plain Go HTTP clients.

Use an interactive Playwright session (you solve the challenge in the opened browser window; no auto-bypass):

```sh
./ordercli foodora login --email you@example.com --password-stdin --browser
```

Prereqs: `node` + `npx` available. First run may download Playwright + Chromium.

Tip: use a persistent profile to keep browser cookies/storage between runs (reduces re-challenges):

```sh
./ordercli foodora login --email you@example.com --password-stdin --browser --browser-profile "$HOME/Library/Application Support/ordercli/browser-profile"
```

### Import cookies from Chrome (no browser run)

If you already solved bot protection / logged in in Chrome, you can import the cookies for the current `base_url` host:

```sh
./ordercli foodora cookies chrome --profile "Default"
./ordercli foodora orders
```

If the bot cookies live on the website domain (e.g. `https://www.foodora.at/`), import from there and store them for the API host:

```sh
./ordercli foodora cookies chrome --url https://www.foodora.at/ --profile "Default"
```

If you have multiple profiles, try `--profile "Profile 1"` (or pass a profile path / Cookies DB via `--cookie-path`).

### Import session from Chrome (no password)

If you’re logged in on the website in Chrome, you can import `refresh_token` + `device_token` and then refresh to an API access token:

```sh
./ordercli foodora session chrome --url https://www.foodora.at/ --profile "Default"
./ordercli foodora session refresh --client-id android
./ordercli foodora history
```
If `session refresh` errors with “refresh token … not found”, that site session isn’t valid for your configured `base_url` (common for some regions).

### Orders

```sh
./ordercli foodora orders
./ordercli foodora orders --watch
./ordercli foodora history
./ordercli foodora history --limit 50
./ordercli foodora history show <orderCode>
./ordercli foodora order <orderCode>
```

### Reorder (add to cart)

Safe default (preview only):

```sh
./ordercli foodora reorder <orderCode>
```

Actually call `orders/{orderCode}/reorder` (adds to cart; does not place an order):

```sh
./ordercli foodora reorder <orderCode> --confirm
```

If you have multiple saved addresses, you must pick one:

```sh
./ordercli foodora reorder <orderCode> --confirm --address-id <id>
```

## deliveroo (WIP)

Requires a valid bearer token (no bypass):

```sh
export DELIVEROO_BEARER_TOKEN='...'
./ordercli deliveroo history
./ordercli deliveroo orders
```

## Safety

This talks to private APIs. Use at your own risk; rate limits / bot protection may block requests.

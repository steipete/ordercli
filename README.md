# foodoracli

Go CLI: login to foodora + show active order status.

Status: early prototype. API details reverse-engineered from an Android XAPK (`25.44.0`).

## Build

```sh
go test ./...
go build ./cmd/foodoracli
```

## Configure country / base URL

Bundled presets (from the APK):

```sh
./foodoracli countries
./foodoracli config set --country HU
./foodoracli config set --country AT
./foodoracli config show
```

Manual:

```sh
./foodoracli config set --base-url https://hu.fd-api.com/api/v5/ --global-entity-id NP_HU --target-iso HU
```

## Login

`oauth2/token` needs a `client_secret` (the app fetches it via remote config). `foodoracli` auto-fetches it on first use and caches it locally.

Optional override (keeps secrets out of shell history):

```sh
export FOODORA_CLIENT_SECRET='...'
./foodoracli login --email you@example.com --password-stdin
```

If MFA triggers, `foodoracli` stores the MFA token locally and prints a safe retry command. Rerun with `--otp <CODE>`.

### Client headers

Some regions (e.g. Austria/mjam `mj.fd-api.com`) expect app-style headers like `X-FP-API-KEY` / `App-Name` / app `User-Agent`. `foodoracli` uses an app-like header profile for `AT` by default.

For corporate flows, you can override the OAuth `client_id`:

```sh
./foodoracli login --email you@example.com --client-id corp_android --password-stdin
```

### Cloudflare / bot protection

Some regions (e.g. Austria/mjam `mj.fd-api.com`) may return Cloudflare HTML (`HTTP 403`) for plain Go HTTP clients.

Use an interactive Playwright session (you solve the challenge in the opened browser window; no auto-bypass):

```sh
./foodoracli login --email you@example.com --password-stdin --browser
```

Prereqs: `node` + `npx` available. First run may download Playwright + Chromium.

## Orders

```sh
./foodoracli orders
./foodoracli orders --watch
./foodoracli order <orderCode>
```

## Safety

This talks to private APIs. Use at your own risk; rate limits / bot protection may block requests.

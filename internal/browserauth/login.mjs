import fs from 'node:fs';
import { chromium } from 'playwright';

const outputPath =
  process.env.ORDERCLI_OUTPUT_PATH ||
  process.env.FOODCLI_OUTPUT_PATH ||
  process.env.FOODORACLI_OUTPUT_PATH;
if (!outputPath) {
  process.stderr.write('ORDERCLI_OUTPUT_PATH missing\n');
  process.exit(2);
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function readStdinJSON() {
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  const raw = Buffer.concat(chunks).toString('utf8').trim();
  if (!raw) throw new Error('stdin empty');
  return JSON.parse(raw);
}

function oauthURL(baseURL) {
  const u = new URL(baseURL);
  if (!u.pathname.endsWith('/')) u.pathname += '/';
  u.pathname += 'oauth2/token';
  return u.toString();
}

function isHTML(status, headers, body) {
  if (status !== 403 && status !== 429 && status !== 503) return false;
  const ct = (headers['content-type'] || '').toLowerCase();
  if (ct.includes('text/html')) return true;
  const b = (body || '').trimStart();
  return b.startsWith('<!DOCTYPE html') || b.startsWith('<html');
}

function tryParseJSON(body) {
  try {
    return JSON.parse(body);
  } catch {
    return null;
  }
}

function isPerimeterXBlocked(status, headers, body) {
  if (status !== 403) return false;
  const ct = (headers['content-type'] || '').toLowerCase();
  if (!ct.includes('application/json')) return false;
  const obj = tryParseJSON(body);
  return !!(obj && (obj.appId || obj.app_id) && (obj.blockScript || obj.altBlockScript));
}

function perimeterXBlockURL(baseURL, body) {
  const obj = tryParseJSON(body);
  if (!obj) return '';
  const baseOrigin = new URL(baseURL).origin;
  const rel = obj.blockScript;
  if (typeof rel === 'string' && rel.startsWith('/')) {
    return new URL(rel, baseOrigin).toString();
  }
  const alt = obj.altBlockScript;
  if (typeof alt === 'string' && alt.startsWith('http')) {
    return alt;
  }
  return '';
}

function perimeterXHTML(baseURL, body) {
  const origin = new URL(baseURL).origin;
  const scriptURL = perimeterXBlockURL(baseURL, body);
  if (!scriptURL) return '';
  return `<!doctype html>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>ordercli — verification</title>
<style>
  :root { color-scheme: dark; }
  body { margin: 0; font: 14px/1.5 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; background: #0b0f14; color: #e8eef7; }
  header { padding: 16px 18px; border-bottom: 1px solid rgba(232,238,247,0.10); }
  h1 { margin: 0; font-size: 14px; font-weight: 650; letter-spacing: .2px; }
  p { margin: 10px 0 0; color: rgba(232,238,247,0.80); }
  main { padding: 18px; }
  .box { border: 1px solid rgba(232,238,247,0.12); border-radius: 12px; padding: 16px; background: rgba(255,255,255,0.03); }
  code { color: #9ad1ff; }
</style>
<base href="${origin}/" />
<header>
  <h1>ordercli verification</h1>
  <p>Complete the verification below. Once cleared, ordercli will continue automatically.</p>
</header>
<main>
  <div class="box">
    <p>Domain: <code>${origin}</code></p>
    <p>If this stays blank, open <code>${origin}</code> in the same window and try again.</p>
  </div>
  <script src="${scriptURL}"></script>
</main>
`;
}

const input = await readStdinJSON();
const timeoutMillis = Math.max(10_000, Number(input.timeout_millis || 0));
const deadline = Date.now() + timeoutMillis;

let browser = null;
let context = null;
if (input.profile_dir) {
  // Persistent profile: keeps cookies/storage between runs.
  context = await chromium.launchPersistentContext(input.profile_dir, { headless: false });
} else {
  browser = await chromium.launch({ headless: false });
  context = await browser.newContext();
}
const page = await context.newPage();

try {
  // Keep the visible tab clean; we’ll render a helpful “verification” page only when blocked.
  await page.goto('about:blank', { waitUntil: 'domcontentloaded' }).catch(() => {});

  const url = oauthURL(input.base_url);
  const origin = new URL(input.base_url).origin;

  let lastLog = 0;
  // Loop until the oauth call stops returning challenge responses (user solved the check), or timeout.
  while (Date.now() < deadline) {
    const res = await context.request.post(url, {
      form: {
        username: input.email,
        password: input.password,
        grant_type: 'password',
        client_secret: input.client_secret,
        scope: 'API_CUSTOMER',
        client_id: input.client_id || 'android',
      },
      headers: {
        Accept: 'application/json',
        'X-Device': input.device_id,
        'X-OTP-Method': input.otp_method || 'sms',
        ...(input.otp_code ? { 'X-OTP': input.otp_code } : {}),
        ...(input.mfa_token ? { 'X-Mfa-Token': input.mfa_token } : {}),
      },
    });

    const status = res.status();
    const headers = res.headers();
    const body = await res.text();

    if (isHTML(status, headers, body) || isPerimeterXBlocked(status, headers, body)) {
      if (Date.now() - lastLog > 5000) {
        lastLog = Date.now();
        process.stderr.write('waiting for browser clearance (solve the challenge in the opened window)...\n');
      }

      if (isPerimeterXBlocked(status, headers, body)) {
        const html = perimeterXHTML(input.base_url, body);
        if (html) {
          await page.setContent(html, { waitUntil: 'domcontentloaded' }).catch(() => {});
        } else {
          await page.goto(origin, { waitUntil: 'domcontentloaded' }).catch(() => {});
        }
      } else {
        await page.goto(origin, { waitUntil: 'domcontentloaded' }).catch(() => {});
      }

      await sleep(1500);
      continue;
    }

    const cookies = await context.cookies(new URL(input.base_url).origin);
    const cookieHeader = cookies.map((c) => `${c.name}=${c.value}`).join('; ');
    const userAgent = await page.evaluate(() => navigator.userAgent).catch(() => '');

    fs.writeFileSync(
      outputPath,
      JSON.stringify({
        status,
        body,
        headers,
        cookie_header: cookieHeader,
        user_agent: userAgent,
      }),
      'utf8',
    );
    await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    process.exit(0);
  }

  process.stderr.write('timeout waiting for browser clearance\n');
  await context.close().catch(() => {});
  if (browser) await browser.close().catch(() => {});
  process.exit(3);
} catch (e) {
  try {
    await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
  } catch {}
  process.stderr.write(String(e?.stack || e) + '\n');
  process.exit(1);
}

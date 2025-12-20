import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { existsSync } from 'node:fs';
import { setTimeout as sleep } from 'node:timers/promises';

function outputPath() {
  return (
    process.env.ORDERCLI_OUTPUT_PATH ||
    process.env.FOODCLI_OUTPUT_PATH ||
    process.env.FOODORACLI_OUTPUT_PATH ||
    ''
  );
}

async function writeOutput(obj) {
  const out = outputPath();
  if (!out) throw new Error('ORDERCLI_OUTPUT_PATH missing');
  await fs.writeFile(out, JSON.stringify(obj), { encoding: 'utf8' });
}

function expandPath(input) {
  if (input.startsWith('~/')) return path.join(os.homedir(), input.slice(2));
  return path.isAbsolute(input) ? input : path.resolve(process.cwd(), input);
}

async function fileExists(candidate) {
  try {
    const stat = await fs.stat(candidate);
    return stat.isFile();
  } catch {
    return false;
  }
}

function looksLikePath(value) {
  return value.includes('/') || value.includes('\\');
}

async function ensureCookieFile(inputPath) {
  const expanded = expandPath(inputPath);
  const stat = await fs.stat(expanded).catch(() => null);
  if (!stat) {
    throw new Error(`Unable to locate Chrome cookie DB at ${expanded}`);
  }
  if (stat.isDirectory()) {
    const directFile = path.join(expanded, 'Cookies');
    if (await fileExists(directFile)) return directFile;
    const networkFile = path.join(expanded, 'Network', 'Cookies');
    if (await fileExists(networkFile)) return networkFile;
    throw new Error(`No Cookies DB found under ${expanded}`);
  }
  return expanded;
}

async function defaultProfileRoot() {
  const candidates = [];
  if (process.platform === 'darwin') {
    candidates.push(
      path.join(os.homedir(), 'Library', 'Application Support', 'Google', 'Chrome'),
      path.join(os.homedir(), 'Library', 'Application Support', 'Microsoft Edge'),
      path.join(os.homedir(), 'Library', 'Application Support', 'Chromium'),
    );
  } else if (process.platform === 'linux') {
    candidates.push(
      path.join(os.homedir(), '.config', 'google-chrome'),
      path.join(os.homedir(), '.config', 'microsoft-edge'),
      path.join(os.homedir(), '.config', 'chromium'),
      path.join(os.homedir(), 'snap', 'chromium', 'common', 'chromium'),
      path.join(os.homedir(), 'snap', 'chromium', 'current', 'chromium'),
    );
  } else if (process.platform === 'win32') {
    const localAppData = process.env.LOCALAPPDATA ?? path.join(os.homedir(), 'AppData', 'Local');
    candidates.push(
      path.join(localAppData, 'Google', 'Chrome', 'User Data'),
      path.join(localAppData, 'Microsoft', 'Edge', 'User Data'),
      path.join(localAppData, 'Chromium', 'User Data'),
    );
  } else {
    throw new Error(`Unsupported platform: ${process.platform}`);
  }

  for (const candidate of candidates) {
    if (existsSync(candidate)) return candidate;
  }
  return candidates[0];
}

async function resolveCookieFile({ profile, explicitCookiePath }) {
  if (explicitCookiePath && explicitCookiePath.trim().length > 0) {
    return ensureCookieFile(explicitCookiePath);
  }
  if (profile && looksLikePath(profile)) {
    return ensureCookieFile(profile);
  }
  const profileName = profile && profile.trim().length > 0 ? profile.trim() : 'Default';
  const baseDir = await defaultProfileRoot();
  return ensureCookieFile(path.join(baseDir, profileName));
}

async function ensureCookiesDirForFallback(cookieFile) {
  const dir = await fs.mkdtemp(path.join(os.tmpdir(), 'ordercli-cookies-secure-'));
  const target = path.join(dir, 'Cookies');
  try {
    await fs.copyFile(cookieFile, target);
  } catch {
    // Best-effort: chrome-cookies-secure might still open the original file.
    return path.dirname(cookieFile);
  }
  return dir;
}

async function settleWithTimeout(promise, timeoutMs, message) {
  let timer = null;
  try {
    return await Promise.race([
      promise,
      new Promise((_, reject) => {
        timer = setTimeout(() => reject(new Error(message)), timeoutMs);
      }),
    ]);
  } finally {
    if (timer) clearTimeout(timer);
  }
}

async function main() {
  const raw = await readStdin();
  const input = raw ? JSON.parse(raw) : {};
  const timeoutMs = Number.isFinite(input.timeout_millis) && input.timeout_millis > 0 ? input.timeout_millis : 5000;
  const targetUrl = String(input.target_url || '').trim();
  if (!targetUrl) throw new Error('target_url missing');

  const cookieFile = await resolveCookieFile({
    profile: input.chrome_profile ? String(input.chrome_profile) : '',
    explicitCookiePath: input.explicit_cookie_path ? String(input.explicit_cookie_path) : '',
  });
  const cookiesDir = await ensureCookiesDirForFallback(cookieFile);

  const mod = await import('chrome-cookies-secure');
  const chromeCookies = mod?.default ?? mod;

  const filterNames = Array.isArray(input.filter_names) ? new Set(input.filter_names.map((v) => String(v))) : null;

  const cookies = await settleWithTimeout(
    chromeCookies.getCookiesPromised(targetUrl, 'puppeteer', cookiesDir),
    timeoutMs,
    `Timed out reading Chrome cookies (after ${timeoutMs} ms)`,
  );

  const pairs = [];
  const seen = new Set();
  if (Array.isArray(cookies)) {
    for (const c of cookies) {
      const name = c?.name ? String(c.name) : '';
      if (!name) continue;
      if (filterNames && filterNames.size > 0 && !filterNames.has(name)) continue;
      if (seen.has(name)) continue;
      const value = c?.value ? String(c.value) : '';
      if (!value) continue;
      seen.add(name);
      pairs.push(`${name}=${value}`);
    }
  }

  await writeOutput({
    cookie_header: pairs.join('; '),
    cookie_count: pairs.length,
    error: '',
  });

  // give fs a beat in case the parent kills us on timeout
  await sleep(10);
}

async function readStdin() {
  return await new Promise((resolve, reject) => {
    let data = '';
    process.stdin.setEncoding('utf8');
    process.stdin.on('data', (chunk) => {
      data += chunk;
    });
    process.stdin.on('end', () => resolve(data));
    process.stdin.on('error', reject);
  });
}

main().catch(async (err) => {
  const message = err instanceof Error ? err.message : String(err);
  try {
    await writeOutput({ cookie_header: '', cookie_count: 0, error: message });
  } catch {
    // ignore
  }
  process.exitCode = 1;
});

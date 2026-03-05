import fs from 'node:fs';
import { chromium } from 'playwright';

const outputPath = process.env.ORDERCLI_OUTPUT_PATH;
if (!outputPath) {
  process.stderr.write('ORDERCLI_OUTPUT_PATH missing\n');
  process.exit(2);
}

async function readStdinJSON() {
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  const raw = Buffer.concat(chunks).toString('utf8').trim();
  if (!raw) throw new Error('stdin empty');
  return JSON.parse(raw);
}

const input = await readStdinJSON();
const browser = await chromium.launch({ headless: input.headless !== false });
const page = await browser.newPage();

try {
  await page.goto(input.url, {
    waitUntil: 'domcontentloaded',
    timeout: Math.max(10_000, Number(input.timeout_millis || 0)),
  });
  await page.waitForLoadState('networkidle', { timeout: 15_000 }).catch(() => {});
  const body = await page.locator('body').innerText();

  fs.writeFileSync(
    outputPath,
    JSON.stringify({
      final_url: page.url(),
      title: await page.title(),
      text: body,
    }),
    'utf8',
  );
  await browser.close();
  process.exit(0);
} catch (err) {
  try {
    await browser.close();
  } catch {}
  process.stderr.write(String(err?.stack || err) + '\n');
  process.exit(1);
}

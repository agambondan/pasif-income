import fs from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { type Page } from '@playwright/test';

const CANDIDATES = [
  process.env.CHROMIUM_BINARY,
  process.env.GOOGLE_CHROME_BIN,
  process.env.CHROME_BIN,
  'chromium',
  'chromium-browser',
  'google-chrome',
  'google-chrome-stable',
  '/usr/bin/chromium',
  '/usr/bin/chromium-browser',
  '/usr/bin/google-chrome',
  '/usr/bin/google-chrome-stable',
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
];

export function resolveChromiumBinary(): string {
  for (const candidate of CANDIDATES) {
    if (!candidate || candidate.trim() === '') {
      continue;
    }
    const trimmed = candidate.trim();
    if (path.isAbsolute(trimmed) && fs.existsSync(trimmed)) {
      return trimmed;
    }
    const lookup = spawnSync('which', [trimmed], { encoding: 'utf8' });
    if (lookup.status === 0) {
      const resolved = lookup.stdout.trim();
      if (resolved) {
        return resolved;
      }
    }
  }
  return '';
}

export async function loginAsAdmin(page: Page): Promise<void> {
  await page.goto('/login');
  const user = await page.evaluate(async () => {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username: 'admin', password: 'admin123' }),
    });
    const text = await res.text();
    if (!res.ok) {
      throw new Error(`login failed ${res.status}: ${text}`);
    }
    return JSON.parse(text) as { id: number; username: string };
  });

  await page.evaluate((value) => {
    localStorage.setItem('user', JSON.stringify(value));
  }, user);

  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('networkidle').catch(() => undefined);
}

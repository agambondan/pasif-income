import { defineConfig } from '@playwright/test';
import { resolveChromiumBinary } from './tests/helpers';

const baseURL = process.env.WEB_BASE_URL ?? 'http://127.0.0.1:13102';
const executablePath = resolveChromiumBinary();

if (!executablePath) {
  throw new Error('No Chromium/Chrome binary found. Set CHROMIUM_BINARY, GOOGLE_CHROME_BIN, or CHROME_BIN.');
}

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  retries: 0,
  timeout: 60_000,
  expect: {
    timeout: 10_000,
  },
  use: {
    baseURL,
    headless: true,
    trace: 'retain-on-failure',
    launchOptions: {
      executablePath,
      args: ['--no-sandbox', '--disable-dev-shm-usage'],
    },
  },
});

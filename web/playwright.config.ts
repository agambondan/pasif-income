import { defineConfig } from '@playwright/test';
import { resolveChromiumBinary } from './tests/helpers';

const baseURL = process.env.WEB_BASE_URL ?? 'http://localhost:13100';
const executablePath = resolveChromiumBinary();

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
    ...(executablePath ? {
      launchOptions: {
        executablePath,
        args: ['--no-sandbox', '--disable-dev-shm-usage'],
      },
    } : {}),
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
});

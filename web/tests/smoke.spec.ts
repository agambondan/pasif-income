import { expect, test } from '@playwright/test';
import { loginAsAdmin } from './helpers';

test('dashboard and integrations smoke', async ({ page }) => {
  await loginAsAdmin(page);

  const storedUser = await page.evaluate(() => localStorage.getItem('user'));
  expect(storedUser).toContain('admin');

  const accountsResponse = await page.evaluate(async () => {
    const res = await fetch('/api/accounts', { credentials: 'include' });
    return {
      status: res.status,
      body: await res.text(),
    };
  });

  expect(accountsResponse.status).toBe(200);
  expect(accountsResponse.body).toContain('youtube');

  await page.goto('/integrations');
  await expect(page).toHaveURL(/\/integrations$/);

  const platformsResponse = await page.evaluate(async () => {
    const res = await fetch('/api/platforms', { credentials: 'include' });
    return {
      status: res.status,
      body: await res.text(),
    };
  });

  expect(platformsResponse.status).toBe(200);
  expect(platformsResponse.body).toContain('youtube');
});

import { test, expect } from '@playwright/test';

test.describe('E2E Production Flow', () => {
  test.beforeEach(async ({ page }) => {
    // Shared login for all tests in this describe block
    await page.goto('/login');
    await page.fill('input[placeholder="admin"]', 'admin');
    await page.fill('input[placeholder="••••••••"]', 'admin123');
    await page.click('button:has-text("ESTABLISH CONNECTION")');
    await expect(page).toHaveURL('/', { timeout: 15000 });
  });

  test('Successful job submission', async ({ page }) => {
    await expect(page.locator('h3:has-text("New Production Job")')).toBeVisible();

    // Fill Production Form
    await page.fill('input[placeholder="stoicism"]', 'QA Test Niche');
    await page.fill('input[placeholder="how to control your mind"]', 'Automation Testing 101');
    
    // Select Account if available
    const distributionMatrix = page.locator('span:has-text("Distribution Matrix")');
    if (await distributionMatrix.isVisible()) {
        const firstAccount = page.locator('label.cursor-pointer').first();
        if (await firstAccount.isVisible()) {
            await firstAccount.click();
        }
    }

    // Intercept the API call to ensure it finishes
    const responsePromise = page.waitForResponse(response => 
      response.url().includes('/api/generate') && response.request().method() === 'POST'
    );

    // Execute Production
    await page.click('button:has-text("EXECUTE PRODUCTION")');
    
    // Wait for the API response
    const response = await responsePromise;
    expect([200, 202]).toContain(response.status());

    const job = await response.json();
    await expect(page.getByText(`Job ${job.id} queued for QA Test Niche: Automation Testing 101`)).toBeVisible({
      timeout: 10000,
    });
  });

  test('Navigate to Archive and Integrations', async ({ page }) => {
    // Navigate to Video Archive
    await page.goto('/videos');
    // Using a more specific selector to avoid strict mode violation (header vs page title)
    await expect(page.locator('h2.text-4xl:has-text("VIDEO ARCHIVE")')).toBeVisible();
    
    // Navigate to Integrations
    await page.goto('/integrations');
    // Use the 4xl version which is the page title, not the sidebar/nav header
    await expect(page.locator('h2.text-4xl:has-text("INTEGRATIONS")')).toBeVisible();
    await expect(page.locator('h3:has-text("YouTube")')).toBeVisible();
  });
});

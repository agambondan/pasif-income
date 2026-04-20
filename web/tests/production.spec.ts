import { test, expect } from '@playwright/test';

test.describe('E2E Production Flow', () => {
  test.beforeEach(async ({ page }) => {
    // Shared login for all tests in this describe block
    await page.goto('/login');
    await page.fill('input[placeholder="admin"]', 'admin');
    await page.fill('input[placeholder="••••••••"]', 'admin123');
    await page.click('button:has-text("ESTABLISH CONNECTION")', { force: true });
    await expect(page).toHaveURL('/', { timeout: 15000 });
  });

  test('Failed login with invalid credentials (MVP-NEG-01)', async ({ page, context }) => {
    // We use a new context to avoid sharing session with beforeEach login
    const newPage = await context.newPage();
    await newPage.goto('/login');
    await newPage.fill('input[placeholder="admin"]', 'admin');
    await newPage.fill('input[placeholder="••••••••"]', 'wrongpassword');
    await newPage.click('button:has-text("ESTABLISH CONNECTION")', { force: true });

    // Verify error message appearance
    await expect(newPage.locator('[data-testid="login-error"]')).toContainText('Invalid credentials');
    await expect(newPage).toHaveURL('/login');
  });

  test('Form validation: Empty topic submission (MVP-NEG-03)', async ({ page }) => {
    await expect(page.locator('h3:has-text("New Production Job")')).toBeVisible();

    await page.locator('#niche-input').fill('Test Niche');
    await page.locator('#topic-input').clear();
    await page.locator('#topic-input').fill(''); // Double-tap just in case
    
    // Attempt to execute
    await page.click('button:has-text("EXECUTE PRODUCTION")', { force: true });
    
    // Check for validation error
    await expect(page.getByTestId('status-message')).toContainText('Topic is required');
    
    // Check if job was NOT submitted (no success message containing 'Job')
    await expect(page.getByTestId('status-message')).not.toContainText(/Job .* queued/);
  });

  test('Successful job submission', async ({ page }) => {
    await expect(page.locator('h3:has-text("New Production Job")')).toBeVisible();
    
    // Fill Production Form
    await page.locator('#niche-input').fill('QA Test Niche');
    await page.locator('#topic-input').fill('Automation Testing 101');
    
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

    // Execute Production - Force click to bypass dev overlay if present
    await page.click('button:has-text("EXECUTE PRODUCTION")', { force: true });
    
    // Wait for the API response
    const response = await responsePromise;
    expect([200, 202]).toContain(response.status());

    const job = await response.json();
    // Use a more specific locator for the success message to avoid strict mode violation
    await expect(page.locator('[data-testid="status-message"]')).toContainText(`Job ${job.id} queued`);
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

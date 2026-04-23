import { expect, test } from '@playwright/test';
import { loginAsAdmin } from './helpers';

test.describe('E2E Production Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      const now = new Date().toISOString();
      const state = {
        jobCounter: 1,
        jobs: [] as Array<{
          id: string;
          niche: string;
          topic: string;
          title?: string;
          description?: string;
          pin_comment?: string;
          video_path?: string;
          status: 'queued' | 'running' | 'completed' | 'failed';
          current_stage: string;
          progress_pct: number;
          error?: string;
          scheduled_at?: string | null;
          created_at: string;
          updated_at: string;
        }>,
        distributions: new Map<string, Array<{
          id: number;
          generation_job_id: string;
          account_id: string;
          platform: string;
          status: 'pending' | 'uploading' | 'completed' | 'failed';
          status_detail: string;
          external_id: string;
          error: string;
          scheduled_at?: string | null;
          created_at: string;
          updated_at: string;
        }>>(),
      };

      const readyAccount = {
        id: 'acc-youtube-qa',
        platform_id: 'youtube',
        display_name: 'YouTube QA Browser',
        auth_method: 'chromium_profile',
        browser_status: 'ready',
        email: 'qa@example.com',
        profile_path: '/app/chromium-profiles/youtube/qa_at_example_com',
        expiry: now,
        created_at: now,
      };

      const voiceType = {
        id: 'en-US-Standard-A',
        label: 'English (US)',
        language: 'English',
        tld: 'us',
      };

      const jobsResponse = () => JSON.stringify(state.jobs);

      const distribute = (jobId: string) => {
        const createdAt = new Date().toISOString();
        state.distributions.set(jobId, [
          {
            id: 1,
            generation_job_id: jobId,
            account_id: readyAccount.id,
            platform: readyAccount.platform_id,
            status: 'pending',
            status_detail: 'queued for publish',
            external_id: '',
            error: '',
            scheduled_at: null,
            created_at: createdAt,
            updated_at: createdAt,
          },
        ]);
      };

      const realFetch = window.fetch.bind(window);
      window.fetch = async (input, init) => {
        const requestUrl = typeof input === 'string' ? input : input.url;
        const parsedUrl = new URL(requestUrl, window.location.origin);
        const method = (init?.method || 'GET').toUpperCase();

        if (parsedUrl.pathname === '/api/auth/login' && method === 'POST') {
          return new Response(JSON.stringify({ id: 1, username: 'admin' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        if (parsedUrl.pathname === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, username: 'admin' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        if (parsedUrl.pathname === '/api/health') {
          return new Response(JSON.stringify({ status: 'ok' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        if (parsedUrl.pathname === '/api/accounts') {
          return new Response(JSON.stringify([readyAccount]), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        if (parsedUrl.pathname === '/api/voice-types') {
          return new Response(JSON.stringify([voiceType]), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        if (parsedUrl.pathname === '/api/jobs' && method === 'GET') {
          return new Response(jobsResponse(), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        if (parsedUrl.pathname === '/api/generate' && method === 'POST') {
          const payload = init?.body ? (JSON.parse(String(init.body)) as {
            niche: string;
            topic: string;
            voice_type: string;
            destinations: Array<{ platform: string; account_id: string }>;
            schedule_mode: string;
            drip_interval_days: number;
          }) : null;

          const jobId = `job-${String(state.jobCounter).padStart(3, '0')}`;
          state.jobCounter += 1;

          const createdAt = new Date().toISOString();
          const job = {
            id: jobId,
            niche: payload?.niche || 'qa-browser',
            topic: payload?.topic || 'generate a short faceless video',
            title: 'QA Browser Generated Video',
            description: 'Browser QA generated video job',
            pin_comment: 'Pinned from browser QA',
            video_path: '/tmp/qa-browser-video.mp4',
            status: 'queued' as const,
            current_stage: payload?.schedule_mode === 'drip_feed' ? 'queued for drip feed' : 'queued for generation',
            progress_pct: 0,
            error: '',
            scheduled_at: null,
            created_at: createdAt,
            updated_at: createdAt,
          };

          state.jobs = [job, ...state.jobs];
          distribute(jobId);

          return new Response(JSON.stringify(job), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        if (parsedUrl.pathname.match(/^\/api\/jobs\/[^/]+\/distributions$/)) {
          const jobId = parsedUrl.pathname.split('/')[3];
          return new Response(JSON.stringify(state.distributions.get(jobId) || []), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        return realFetch(input, init);
      };
    });

    await loginAsAdmin(page);
  });

  test('queues a production job from the dashboard', async ({ page }) => {
    await expect(page).toHaveURL(/\/$/);
    await expect(page.getByRole('heading', { name: 'Production cockpit' })).toBeVisible();

    await page.getByPlaceholder('stoicism').fill('qa-browser');
    await page.getByPlaceholder('how to control your mind').fill('Generate a short faceless video from the browser QA flow.');

    const accountCheckbox = page.locator('input[type="checkbox"]').first();
    await expect(accountCheckbox).toHaveCount(1);
    await accountCheckbox.click({ force: true });
    await expect(accountCheckbox).toBeChecked();

    const generateButton = page.getByRole('button', { name: 'EXECUTE PRODUCTION' });
    await expect(generateButton).toBeEnabled();

    await generateButton.click();
    await expect(page.getByText('Job job-001 queued')).toBeVisible();
    await expect(page.getByText('qa-browser').first()).toBeVisible();
    await expect(page.getByText('Generate a short faceless video from the browser QA flow.').first()).toBeVisible();
    await expect(page.getByText('Distribution Jobs')).toBeVisible();
    await expect(page.getByText('Pending / Uploading')).toBeVisible();
    await expect(page.getByText('queued for publish')).toBeVisible();
  });
});

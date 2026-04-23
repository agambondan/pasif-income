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

test('integrations page shows launch feedback for chromium profiles', async ({ page }) => {
  await page.addInitScript(() => {
    const account = {
      id: 'youtube-qa-browser-test',
      platform_id: 'youtube',
      display_name: 'YouTube QA Browser',
      auth_method: 'chromium_profile',
      email: 'qa-browser@local',
      profile_path: '/data/browser_profiles/youtube/qa-browser_at_local',
      browser_status: 'needs_login',
      expiry: new Date().toISOString(),
      created_at: new Date().toISOString(),
    };
    let statusCalls = 0;
    const realFetch = window.fetch.bind(window);
    window.fetch = async (input, init) => {
      const requestUrl = typeof input === 'string' ? input : input.url;
      const parsedUrl = new URL(requestUrl, window.location.origin);

      if (parsedUrl.pathname === '/api/auth/login' && init?.method === 'POST') {
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

      if (parsedUrl.pathname === '/api/platforms') {
        return new Response(JSON.stringify([
          { id: 'youtube', name: 'YouTube', supported_methods: ['api', 'chromium_profile'], description: 'Video platform' },
        ]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      if (parsedUrl.pathname === `/api/accounts/${account.id}/launch`) {
        return new Response(JSON.stringify({ id: account.id, status: 'queued' }), {
          status: 202,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      if (parsedUrl.pathname === `/api/accounts/${account.id}/status`) {
        statusCalls += 1;
        const updated = {
          ...account,
          browser_status: statusCalls >= 2 ? 'ready' : 'needs_login',
        };
        Object.assign(account, updated);
        return new Response(JSON.stringify(updated), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      if (parsedUrl.pathname === '/api/accounts') {
        return new Response(JSON.stringify([account]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      return realFetch(input, init);
    };
  });

  await loginAsAdmin(page);

  await page.goto('/integrations');
  await expect(page).toHaveURL(/\/integrations$/);

  const qaCard = page.getByTestId('browser-account-card').filter({ hasText: 'YouTube QA Browser' });
  await expect(qaCard).toBeVisible();
  await expect(qaCard.getByTestId('browser-launch-status')).toContainText('Click Open Browser');

  await qaCard.getByRole('button', { name: 'Open Browser' }).click();

  await expect(qaCard.getByTestId('browser-launch-status')).toContainText(/(Queued on host launcher|Launched on host|Browser profile is ready)/);
  await expect(qaCard.getByTestId('browser-launch-status')).toContainText('Browser profile is ready');
  await expect(page.getByText('Browser launch queued on host launcher.')).toBeVisible();
});

test('create profile auto-refreshes chromium status', async ({ page }) => {
  await page.addInitScript(() => {
    const browserAccounts: Array<{
      id: string;
      platform_id: string;
      display_name: string;
      auth_method: string;
      email: string;
      profile_path: string;
      browser_status?: string;
      expiry: string;
      created_at: string;
    }> = [];
    let statusCalls = 0;
    const realFetch = window.fetch.bind(window);
    window.fetch = async (input, init) => {
      const requestUrl = typeof input === 'string' ? input : input.url;
      const parsedUrl = new URL(requestUrl, window.location.origin);

      if (parsedUrl.pathname === '/api/auth/login' && init?.method === 'POST') {
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

      if (parsedUrl.pathname === '/api/platforms') {
        return new Response(JSON.stringify([
          { id: 'youtube', name: 'YouTube', supported_methods: ['api', 'chromium_profile'], description: 'Video platform' },
        ]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      if (parsedUrl.pathname === '/api/auth/youtube' && parsedUrl.searchParams.get('method') === 'chromium_profile') {
        const email = parsedUrl.searchParams.get('email') || 'qa-create@local';
        browserAccounts.splice(0, browserAccounts.length, {
          id: 'youtube-created-browser',
          platform_id: 'youtube',
          display_name: 'YOUTUBE Chromium Profile',
          auth_method: 'chromium_profile',
          email,
          profile_path: '/data/browser_profiles/youtube/qa-create_at_local',
          browser_status: 'needs_login',
          expiry: new Date().toISOString(),
          created_at: new Date().toISOString(),
        });
        return new Response(JSON.stringify(browserAccounts[0]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      if (parsedUrl.pathname === '/api/accounts/youtube-created-browser/status') {
        statusCalls += 1;
        browserAccounts[0] = {
          ...browserAccounts[0],
          browser_status: statusCalls >= 2 ? 'ready' : 'needs_login',
        };
        return new Response(JSON.stringify(browserAccounts[0]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      if (parsedUrl.pathname === '/api/accounts') {
        return new Response(JSON.stringify(browserAccounts), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      return realFetch(input, init);
    };
  });

  await loginAsAdmin(page);
  await page.goto('/integrations');
  await expect(page).toHaveURL(/\/integrations$/);

  await page.getByPlaceholder('account@gmail.com').fill('qa-create@local');
  await page.getByRole('button', { name: /CREATE & OPEN PROFILE/ }).click();

  const createdCard = page.getByTestId('browser-account-card').filter({ hasText: 'YOUTUBE Chromium Profile' });
  await expect(createdCard).toBeVisible();
  await expect(createdCard.getByTestId('browser-launch-status')).toContainText(/(Browser profile created|Launched on host|Browser profile is ready)/);
  await expect(createdCard.getByTestId('browser-launch-status')).toContainText('Browser profile is ready');
  await expect(page.getByText('Browser login queued for qa-create@local. Open the host launcher to open Chromium on your desktop.')).toBeVisible();
});

test('videos page renders account comparison leaderboard', async ({ page }) => {
  const metricsPayload = {
    summary: {
      total_videos: 4,
      total_views: 250,
      total_likes: 25,
      total_comments: 10,
      latest_collected_at: '2026-04-22T10:00:00.000Z',
    },
    latest: [],
    history: [
      {
        id: 1,
        user_id: 1,
        generation_job_id: 'job-a',
        distribution_job_id: 10,
        account_id: 'acc-youtube-a',
        platform: 'youtube',
        niche: 'tech',
        external_id: 'yt-a-1',
        video_title: 'Video A1',
        view_count: 100,
        like_count: 10,
        comment_count: 4,
        collected_at: '2026-04-22T09:45:00.000Z',
      },
      {
        id: 2,
        user_id: 1,
        generation_job_id: 'job-b',
        distribution_job_id: 11,
        account_id: 'acc-youtube-a',
        platform: 'youtube',
        niche: 'tech',
        external_id: 'yt-a-2',
        video_title: 'Video A2',
        view_count: 50,
        like_count: 5,
        comment_count: 2,
        collected_at: '2026-04-22T09:50:00.000Z',
      },
      {
        id: 3,
        user_id: 1,
        generation_job_id: 'job-c',
        distribution_job_id: 12,
        account_id: 'acc-youtube-b',
        platform: 'youtube',
        niche: 'tech',
        external_id: 'yt-b-1',
        video_title: 'Video B1',
        view_count: 30,
        like_count: 2,
        comment_count: 1,
        collected_at: '2026-04-22T09:55:00.000Z',
      },
      {
        id: 4,
        user_id: 1,
        generation_job_id: 'job-d',
        distribution_job_id: 13,
        account_id: 'acc-tiktok-c',
        platform: 'tiktok',
        niche: 'tech',
        external_id: 'tt-c-1',
        video_title: 'Video C1',
        view_count: 70,
        like_count: 8,
        comment_count: 3,
        collected_at: '2026-04-22T09:58:00.000Z',
      },
    ],
    alerts: [],
  };

  await page.addInitScript((payload) => {
    const stubbed = {
      '/api/auth/login': {
        id: 1,
        username: 'admin',
      },
      '/api/auth/me': {
        id: 1,
        username: 'admin',
      },
      '/api/videos': ['video-a', 'video-b'],
      '/api/publish/history': [],
      '/api/metrics': payload,
      '/api/community/replies': {
        summary: {
          total: 0,
          drafts: 0,
          replied: 0,
          latest_created_at: null,
        },
        latest: [],
      },
    };

    const realFetch = window.fetch.bind(window);
    window.fetch = async (input, init) => {
      const requestUrl = typeof input === 'string' ? input : input.url;
      const parsedUrl = new URL(requestUrl, window.location.origin);
      const stubbedPayload = stubbed[parsedUrl.pathname as keyof typeof stubbed];
      if (stubbedPayload !== undefined) {
        return new Response(JSON.stringify(stubbedPayload), {
          status: 200,
          headers: {
            'Content-Type': 'application/json',
          },
        });
      }
      return realFetch(input, init);
    };
  }, metricsPayload);

  await loginAsAdmin(page);
  await page.goto('/videos');
  await expect(page).toHaveURL(/\/videos$/);
  await expect(page.getByRole('heading', { name: 'Account Comparison' })).toBeVisible();
  const comparison = page.getByTestId('account-comparison');
  await expect(comparison.getByText('acc-youtube-a')).toBeVisible();
  await expect(comparison.getByText('acc-youtube-b')).toBeVisible();
  await expect(comparison.getByText('acc-tiktok-c')).toBeVisible();
  await expect(comparison.getByText('171 impact')).toBeVisible();
  await expect(comparison.getByText('33 impact')).toBeVisible();
  await expect(comparison.getByText('81 impact')).toBeVisible();
  await expect(page.getByText('No comparison data yet')).toHaveCount(0);
});

test('agent console renders live event stream', async ({ page }) => {
  await page.addInitScript(() => {
    const realFetch = window.fetch.bind(window);
    window.fetch = async (input, init) => {
      const requestUrl = typeof input === 'string' ? input : input.url;
      const parsedUrl = new URL(requestUrl, window.location.origin);

      if (parsedUrl.pathname === '/api/auth/me') {
        return new Response(JSON.stringify({ id: 1, username: 'admin' }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      if (parsedUrl.pathname === '/api/jobs') {
        return new Response(JSON.stringify([
          {
            id: 'job-1',
            niche: 'stoicism',
            topic: 'how to control your mind',
            current_stage: 'running',
            progress_pct: 42,
          },
        ]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      return realFetch(input, init);
    };

    class MockEventSource {
      url;
      onmessage = null;
      onerror = null;

      constructor(url) {
        this.url = url;
        setTimeout(() => {
          if (this.onmessage) {
            this.onmessage({
              data: JSON.stringify({
                id: 'evt-1',
                job_id: new URL(url, window.location.origin).searchParams.get('job_id') || 'job-1',
                type: 'thought',
                content: 'Inspecting the selected clip candidate.',
                timestamp: new Date().toISOString(),
              }),
            });
          }
        }, 50);
      }

      close() {}
    }

    window.EventSource = MockEventSource;
  });

  await page.goto('/agent');
  await expect(page).toHaveURL(/\/agent$/);
  await expect(page.getByRole('heading', { name: 'Gemini Agent Control' })).toBeVisible();
  await expect(page.getByRole('combobox')).toHaveValue('job-1');
  await expect(page.getByText('Inspecting the selected clip candidate.')).toBeVisible();
});

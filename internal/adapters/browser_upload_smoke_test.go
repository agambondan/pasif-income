package adapters

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

const youtubeSmokeHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>Smoke Upload</title>
    <style>
      body { font-family: sans-serif; padding: 24px; }
      .row { margin-bottom: 16px; }
      #complete { display: none; margin-top: 12px; color: #16a34a; }
      #publish { display: none; margin-top: 12px; }
    </style>
  </head>
  <body>
    <div class="row">
      <button id="upload-entry">Upload</button>
    </div>
    <div class="row" id="processing-note">edit video</div>
    <div class="row">
      <input id="upload-file" type="file" accept="video/*" />
    </div>
    <div class="row">
      <textarea id="title-textarea" aria-label="title"></textarea>
    </div>
    <div class="row">
      <textarea id="description-textarea"></textarea>
    </div>
    <div class="row">
      <button id="kids-choice">no, this is not made for kids</button>
    </div>
    <div class="row">
      <button id="next-step">Next</button>
    </div>
    <div class="row">
      <button id="privacy-choice">public</button>
    </div>
    <div id="complete">Upload complete</div>
    <div class="row">
      <button id="publish" type="button">Publish</button>
    </div>
    <script>
      const next = document.getElementById('next-step');
      next.addEventListener('click', () => {
        const current = Number(next.dataset.count || '0') + 1;
        next.dataset.count = String(current);
        next.textContent = 'Next';
      });

      const kids = document.getElementById('kids-choice');
      kids.addEventListener('click', () => {
        kids.dataset.selected = 'true';
      });

      const fileInput = document.getElementById('upload-file');
      fileInput.addEventListener('change', () => {
        document.body.dataset.fileAttached = fileInput.files && fileInput.files.length > 0 ? 'true' : 'false';
      });

      const privacy = document.getElementById('privacy-choice');
      privacy.addEventListener('click', () => {
        privacy.dataset.selected = 'true';
        setTimeout(() => {
          document.getElementById('complete').style.display = 'block';
          document.getElementById('publish').style.display = 'inline-block';
        }, 250);
      });

      document.getElementById('publish').addEventListener('click', () => {
        document.body.dataset.published = 'true';
      });
    </script>
  </body>
</html>`

const tiktokSmokeHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>TikTok Smoke Upload</title>
    <style>
      body { font-family: sans-serif; padding: 24px; }
      .row { margin-bottom: 16px; }
      #post { margin-top: 12px; }
    </style>
  </head>
  <body>
    <div class="row">
      <button id="upload-entry">Upload</button>
    </div>
    <div class="row" id="processing-note">edit video</div>
    <div class="row">
      <input id="upload-file" type="file" accept="video/*" />
    </div>
    <div class="row">
      <input id="title-input" placeholder="title" type="text" />
    </div>
    <div class="row">
      <textarea id="description-input" placeholder="description"></textarea>
    </div>
    <div class="row">
      <button id="post" type="button">Post</button>
    </div>
    <script>
      document.getElementById('upload-entry').addEventListener('click', () => {
        document.body.dataset.uploadOpened = 'true';
      });
      document.getElementById('upload-file').addEventListener('change', () => {
        document.body.dataset.fileAttached = 'true';
      });
      document.getElementById('post').addEventListener('click', () => {
        document.body.dataset.posted = 'true';
      });
    </script>
  </body>
</html>`

const instagramSmokeHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>Instagram Smoke Upload</title>
    <style>
      body { font-family: sans-serif; padding: 24px; }
      .row { margin-bottom: 16px; }
      #share { display: none; margin-top: 12px; }
    </style>
  </head>
  <body>
    <div class="row">
      <button id="create-entry">Create</button>
    </div>
    <div class="row">
      <input id="upload-file" type="file" accept="video/*" />
    </div>
    <div class="row">
      <textarea id="caption-input" placeholder="Write a caption"></textarea>
    </div>
    <div class="row">
      <button id="next-step">Next</button>
    </div>
    <div class="row">
      <button id="share" type="button">Share</button>
    </div>
    <script>
      document.getElementById('create-entry').addEventListener('click', () => {
        document.body.dataset.createOpened = 'true';
      });
      document.getElementById('upload-file').addEventListener('change', () => {
        document.body.dataset.fileAttached = 'true';
      });
      document.getElementById('next-step').addEventListener('click', () => {
        const current = Number(document.body.dataset.nextCount || '0') + 1;
        document.body.dataset.nextCount = String(current);
        if (current >= 2) {
          document.getElementById('share').style.display = 'inline-block';
        }
      });
      document.getElementById('share').addEventListener('click', () => {
        document.body.dataset.shared = 'true';
      });
    </script>
  </body>
</html>`

func TestChromiumRunnerAutomateYouTubeUploadSmoke(t *testing.T) {
	t.Parallel()
	runChromiumSmokeUploadTest(t, "youtube", youtubeSmokeHTML, "Smoke Title", "Smoke Description")
}

func TestChromiumRunnerAutomateTikTokUploadSmoke(t *testing.T) {
	t.Parallel()
	runChromiumSmokeUploadTest(t, "tiktok", tiktokSmokeHTML, "Smoke Title", "Smoke Description")
}

func TestChromiumRunnerAutomateInstagramUploadSmoke(t *testing.T) {
	t.Parallel()
	runChromiumSmokeUploadTest(t, "instagram", instagramSmokeHTML, "Smoke Title", "Smoke Description")
}

func runChromiumSmokeUploadTest(t *testing.T, platformID, smokeHTML, title, description string) {
	t.Helper()

	runner := NewChromiumRunnerFromEnv()
	if runner.binary == "" {
		t.Skip("chromium binary not available")
	}

	dir := t.TempDir()
	profilePath := filepath.Join(dir, platformID+"-profile")
	videoPath := filepath.Join(dir, platformID+"-smoke-video.mp4")
	if err := os.WriteFile(videoPath, []byte("fake-video"), 0o644); err != nil {
		t.Fatalf("write smoke video: %v", err)
	}

	targetURL := "data:text/html;charset=utf-8," + url.PathEscape(smokeHTML)

	probeCtx, probeCancel := context.WithTimeout(context.Background(), 30*time.Second)
	probeOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	probeOpts = append(probeOpts,
		chromedp.ExecPath(runner.binary),
		chromedp.UserDataDir(filepath.Join(dir, platformID+"-probe-profile")),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.WindowSize(1280, 800),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Headless,
	)
	probeAllocCtx, probeAllocCancel := chromedp.NewExecAllocator(probeCtx, probeOpts...)
	probeRunCtx, probeRunCancel := chromedp.NewContext(probeAllocCtx)
	defer probeRunCancel()
	defer probeAllocCancel()
	defer probeCancel()
	defer func() {
		time.Sleep(2 * time.Second)
		_ = os.RemoveAll(filepath.Join(dir, platformID+"-probe-profile"))
	}()

	var probeBody string
	if err := chromedp.Run(probeRunCtx,
		chromedp.Navigate(targetURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Evaluate(`document.body ? document.body.innerText : ''`, &probeBody),
	); err != nil {
		t.Fatalf("probe navigation failed: %v", err)
	}
	t.Logf("%s probe body text: %q", platformID, probeBody)
	if strings.TrimSpace(probeBody) == "" {
		t.Fatalf("probe body text was empty")
	}

	if err := setFirstValue(probeRunCtx, titleSelectors(platformID), "Probe Title"); err != nil {
		t.Fatalf("probe title set failed: %v", err)
	}
	if err := setFirstValue(probeRunCtx, descriptionSelectors(platformID), "Probe Description"); err != nil {
		t.Fatalf("probe description set failed: %v", err)
	}
	if err := clickFirstTextMatch(probeRunCtx, uploadActionTerms(platformID)); err != nil {
		t.Fatalf("probe upload click failed: %v", err)
	}
	if err := clickFirstTextMatch(probeRunCtx, publishActionTerms(platformID)); err != nil {
		t.Fatalf("probe publish click failed: %v", err)
	}

	var titleValue, descriptionValue string
	if err := chromedp.Run(probeRunCtx,
		chromedp.Evaluate(`document.querySelector('textarea, input[type="text"]') ? document.querySelector('textarea, input[type="text"]').value : ''`, &titleValue),
		chromedp.Evaluate(`document.querySelectorAll('textarea, input[type="text"]').length > 1 ? document.querySelectorAll('textarea, input[type="text"]')[1].value : ''`, &descriptionValue),
	); err != nil {
		t.Fatalf("probe value read failed: %v", err)
	}
	t.Logf("%s probe primary value: %q", platformID, titleValue)
	t.Logf("%s probe secondary value: %q", platformID, descriptionValue)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	progress := make([]string, 0, 8)
	err := runner.AutomateUpload(
		ctx,
		profilePath,
		targetURL,
		videoPath,
		title,
		description,
		platformID,
		func(stage string) {
			progress = append(progress, stage)
		},
	)
	if err != nil {
		debugDir := filepath.Join(profilePath, "debug")
		if entries, readErr := os.ReadDir(debugDir); readErr == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				path := filepath.Join(debugDir, entry.Name())
				data, readFileErr := os.ReadFile(path)
				if readFileErr != nil {
					t.Logf("debug artifact %s unreadable: %v", entry.Name(), readFileErr)
					continue
				}
				if strings.HasSuffix(entry.Name(), ".html") {
					limit := len(data)
					if limit > 4000 {
						limit = 4000
					}
					t.Logf("debug artifact %s:\n%s", entry.Name(), string(data[:limit]))
					continue
				}
				t.Logf("debug artifact %s: %d bytes", entry.Name(), len(data))
				if strings.HasSuffix(entry.Name(), ".json") {
					t.Logf("debug artifact %s contents:\n%s", entry.Name(), string(data))
				}
			}
		} else {
			t.Logf("debug artifact directory unreadable: %v", readErr)
		}
		t.Fatalf("%s automation smoke failed: %v (progress=%s)", platformID, err, strings.Join(progress, ","))
	}

	if len(progress) == 0 {
		t.Fatalf("%s smoke expected progress callbacks to be recorded", platformID)
	}

	if _, err := os.Stat(filepath.Join(profilePath, "Default")); err != nil {
		t.Fatalf("expected chromium profile directory to be created: %v", err)
	}

	time.Sleep(2 * time.Second)
	_ = os.RemoveAll(profilePath)
}

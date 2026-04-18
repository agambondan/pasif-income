package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type ChromiumRunner struct {
	binary string
}

func NewChromiumRunnerFromEnv() *ChromiumRunner {
	return &ChromiumRunner{binary: resolveChromiumBinary()}
}

func (r *ChromiumRunner) Open(ctx context.Context, profilePath, targetURL string) error {
	if profilePath == "" {
		return fmt.Errorf("profile path is required")
	}
	if targetURL == "" {
		return fmt.Errorf("target url is required")
	}

	if r.binary == "" {
		return fmt.Errorf("no chromium binary found; set CHROMIUM_BINARY or install chromium/google-chrome")
	}

	if err := os.MkdirAll(profilePath, 0o755); err != nil {
		return err
	}

	args := []string{
		"--user-data-dir=" + profilePath,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-dev-shm-usage",
		"--new-window",
	}
	if headlessEnabled() {
		args = append(args, "--headless=new")
	}
	args = append(args, targetURL)

	log.Printf("Launching Chromium: %s %s\n", r.binary, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, r.binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	waitFor := browserWaitDuration()
	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		_ = killProcess(cmd)
		return ctx.Err()
	case <-timer.C:
		_ = killProcess(cmd)
		return nil
	}
}

func (r *ChromiumRunner) AutomateUpload(ctx context.Context, profilePath, targetURL, filePath, title, description, platformID string, progress func(string)) (err error) {
	if profilePath == "" {
		return fmt.Errorf("profile path is required")
	}
	if targetURL == "" {
		return fmt.Errorf("target url is required")
	}
	if filePath == "" {
		return fmt.Errorf("file path is required")
	}
	if r.binary == "" {
		return fmt.Errorf("no chromium binary found; set CHROMIUM_BINARY or install chromium/google-chrome")
	}

	if err := os.MkdirAll(profilePath, 0o755); err != nil {
		return err
	}

	if progress != nil {
		progress("launching_browser")
	}

	allocOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	allocOpts = append(allocOpts,
		chromedp.ExecPath(r.binary),
		chromedp.UserDataDir(profilePath),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.WindowSize(1440, 960),
		chromedp.Flag("disable-dev-shm-usage", true),
	)
	if headlessEnabled() {
		allocOpts = append(allocOpts, chromedp.Headless)
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, allocOpts...)
	defer allocCancel()

	runCtx, runCancel := chromedp.NewContext(allocCtx)
	defer runCancel()

	timeout := browserAutomationTimeout()
	runCtx, timeoutCancel := context.WithTimeout(runCtx, timeout)
	defer timeoutCancel()

	stage := "navigate"
	defer func() {
		if err == nil {
			return
		}
		if captureErr := captureAutomationArtifacts(runCtx, profilePath, platformID, stage, err); captureErr != nil {
			log.Printf("chromium upload: failed to write debug artifacts: %v\n", captureErr)
		}
	}()

	if progress != nil {
		progress("loading_target")
	}
	if err = chromedp.Run(runCtx,
		chromedp.Navigate(targetURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return err
	}

	stage = "open_upload_flow"
	if progress != nil {
		progress("opening_upload_flow")
	}
	if err = clickFirstTextMatch(runCtx, uploadActionTerms(platformID)); err != nil {
		log.Printf("chromium upload: upload action not found for %s: %v\n", platformID, err)
	}

	stage = "attach_file"
	if progress != nil {
		progress("attaching_file")
	}
	if err = waitAndSetUploadFile(runCtx, platformID, filePath); err != nil {
		return err
	}

	stage = "fill_metadata"
	if progress != nil {
		progress("filling_metadata")
	}
	_ = setFirstValue(runCtx, titleSelectors(platformID), title)
	_ = setFirstValue(runCtx, descriptionSelectors(platformID), description)

	stage = "publish"
	if progress != nil {
		progress("publishing")
	}
	if err = clickFirstTextMatch(runCtx, publishActionTerms(platformID)); err != nil {
		log.Printf("chromium upload: publish action not found for %s: %v\n", platformID, err)
	}

	if progress != nil {
		progress("completed")
	}
	return nil
}

func resolveChromiumBinary() string {
	candidates := []string{
		os.Getenv("CHROMIUM_BINARY"),
		os.Getenv("GOOGLE_CHROME_BIN"),
		os.Getenv("CHROME_BIN"),
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return ""
}

func headlessEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("BROWSER_HEADLESS")))
	if value == "false" || value == "0" {
		return false
	}
	return runtime.GOOS != "darwin" || value != ""
}

func browserWaitDuration() time.Duration {
	raw := strings.TrimSpace(os.Getenv("BROWSER_WAIT_SECONDS"))
	if raw == "" {
		return 5 * time.Second
	}
	duration, err := time.ParseDuration(raw + "s")
	if err != nil {
		return 5 * time.Second
	}
	return duration
}

func killProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := cmd.Process.Kill(); err != nil {
		return err
	}
	_, _ = cmd.Process.Wait()
	return nil
}

func browserTargetURL(platformID string) string {
	switch platformID {
	case "youtube":
		if url := strings.TrimSpace(os.Getenv("YOUTUBE_UPLOAD_URL")); url != "" {
			return url
		}
		return "https://studio.youtube.com"
	case "tiktok":
		if url := strings.TrimSpace(os.Getenv("TIKTOK_UPLOAD_URL")); url != "" {
			return url
		}
		return "https://www.tiktok.com/upload"
	case "instagram":
		if url := strings.TrimSpace(os.Getenv("INSTAGRAM_UPLOAD_URL")); url != "" {
			return url
		}
		return "https://www.instagram.com"
	default:
		return ""
	}
}

func browserProfileMetadataPath(profilePath string) string {
	return filepath.Join(profilePath, "profile.json")
}

func captureAutomationArtifacts(ctx context.Context, profilePath, platformID, stage string, cause error) error {
	debugDir := filepath.Join(profilePath, "debug")
	if err := os.MkdirAll(debugDir, 0o755); err != nil {
		return err
	}

	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	base := sanitizeArtifactName(fmt.Sprintf("%s_%s", platformID, stage))
	basePath := filepath.Join(debugDir, fmt.Sprintf("%s_%s", stamp, base))

	manifest := map[string]any{
		"platform_id": platformID,
		"stage":       stage,
		"error":       cause.Error(),
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}
	if data, err := json.MarshalIndent(manifest, "", "  "); err == nil {
		_ = os.WriteFile(basePath+".json", data, 0o644)
	}

	var html string
	if err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html, chromedp.ByQuery)); err == nil && html != "" {
		_ = os.WriteFile(basePath+".html", []byte(html), 0o644)
	}

	var screenshot []byte
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&screenshot, 90)); err == nil && len(screenshot) > 0 {
		_ = os.WriteFile(basePath+".png", screenshot, 0o644)
	}

	return nil
}

func sanitizeArtifactName(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_", "@", "_", ".", "_")
	cleaned := strings.ToLower(strings.TrimSpace(replacer.Replace(value)))
	if cleaned == "" {
		return "artifact"
	}
	return cleaned
}

func browserAutomationTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("BROWSER_AUTOMATION_TIMEOUT_SECONDS"))
	if raw == "" {
		return 2 * time.Minute
	}
	duration, err := time.ParseDuration(raw + "s")
	if err != nil || duration < 30*time.Second {
		return 2 * time.Minute
	}
	return duration
}

func waitAndSetUploadFile(ctx context.Context, platformID, filePath string) error {
	selectors := uploadFileSelectors(platformID)
	for _, selector := range selectors {
		if err := chromedp.Run(ctx,
			chromedp.WaitVisible(selector, chromedp.ByQuery),
			chromedp.SetUploadFiles(selector, []string{filePath}, chromedp.ByQuery),
		); err == nil {
			return nil
		}
	}
	return fmt.Errorf("upload file input not found")
}

func setFirstValue(ctx context.Context, selectors []string, value string) error {
	for _, selector := range selectors {
		if err := chromedp.Run(ctx,
			chromedp.SetValue(selector, value, chromedp.ByQuery),
			chromedp.Blur(selector, chromedp.ByQuery),
		); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no matching field found")
}

func clickFirstTextMatch(ctx context.Context, terms []string) error {
	for _, term := range terms {
		js := fmt.Sprintf(`(() => {
			const needle = %q;
			const nodes = Array.from(document.querySelectorAll('button, a, tp-yt-paper-button, ytcp-button, div[role="button"], span[role="button"], input[type="button"], input[type="submit"]'));
			const match = nodes.find((el) => ((el.innerText || el.textContent || el.value || '').trim().toLowerCase().includes(needle)));
			if (match) {
				match.click();
				return true;
			}
			return false;
		})()`, strings.ToLower(term))
		var clicked bool
		if err := chromedp.Run(ctx, chromedp.Evaluate(js, &clicked)); err == nil && clicked {
			return nil
		}
	}
	return fmt.Errorf("no matching text action found")
}

func uploadActionTerms(platformID string) []string {
	switch platformID {
	case "youtube":
		return []string{"Create", "Upload videos", "Select files"}
	case "tiktok":
		return []string{"Upload", "Select file", "Choose file"}
	case "instagram":
		return []string{"New post", "Select from computer", "Choose from computer"}
	default:
		return []string{"Upload", "Select file"}
	}
}

func publishActionTerms(platformID string) []string {
	switch platformID {
	case "youtube":
		return []string{"Publish", "Done"}
	case "tiktok":
		return []string{"Post", "Publish"}
	case "instagram":
		return []string{"Share", "Publish"}
	default:
		return []string{"Publish", "Share", "Post"}
	}
}

func titleSelectors(platformID string) []string {
	switch platformID {
	case "youtube":
		return []string{
			"input[aria-label*='title']",
			"textarea[aria-label*='title']",
			"input#title-textarea",
			"textarea#title-textarea",
		}
	case "tiktok":
		return []string{"input[placeholder*='title']", "textarea[placeholder*='title']", "input[type='text']"}
	case "instagram":
		return []string{"textarea[placeholder*='Write a caption']", "textarea[placeholder*='caption']", "textarea"}
	default:
		return []string{"input[aria-label*='title']", "textarea[aria-label*='title']", "textarea"}
	}
}

func descriptionSelectors(platformID string) []string {
	switch platformID {
	case "youtube":
		return []string{
			"textarea[aria-label*='description']",
			"textarea#description-textarea",
			"textarea",
		}
	case "tiktok":
		return []string{"textarea[placeholder*='description']", "textarea"}
	case "instagram":
		return []string{"textarea[placeholder*='caption']", "textarea"}
	default:
		return []string{"textarea[aria-label*='description']", "textarea"}
	}
}

func uploadFileSelectors(platformID string) []string {
	base := []string{
		"input[type=file]",
		"input[name=file]",
	}
	switch platformID {
	case "youtube":
		return append([]string{
			"input[type=file]",
			"input[accept*='video']",
		}, base...)
	case "tiktok":
		return append([]string{
			"input[type=file]",
			"input[accept*='video']",
		}, base...)
	case "instagram":
		return append([]string{
			"input[type=file]",
			"input[accept*='video']",
		}, base...)
	default:
		return base
	}
}

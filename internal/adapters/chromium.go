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
		"--ignore-certificate-errors",
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
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("no-sandbox", true),
	)

	if waylandEnabled() {
		allocOpts = append(allocOpts,
			chromedp.Flag("ozone-platform", "wayland"),
			chromedp.Flag("enable-features", "UseOzonePlatform"),
		)
	}
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

	if platformID == "youtube" {
		if err = automateYouTubeUpload(runCtx, filePath, title, description, progress); err != nil {
			return err
		}
	} else if platformID == "tiktok" {
		if err = automateTikTokUpload(runCtx, filePath, title, description, progress); err != nil {
			return err
		}
	} else if platformID == "instagram" {
		if err = automateInstagramUpload(runCtx, filePath, title, description, progress); err != nil {
			return err
		}
	} else {
		if err = automateGenericUpload(runCtx, platformID, filePath, title, description, progress); err != nil {
			return err
		}
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

func waylandEnabled() bool {
	return os.Getenv("XDG_SESSION_TYPE") == "wayland"
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
	var (
		href       string
		title      string
		bodyText   string
		html       string
		screenshot []byte
	)
	_ = chromedp.Run(ctx,
		chromedp.Evaluate(`window.location.href`, &href),
		chromedp.Evaluate(`document.title || ''`, &title),
		chromedp.Evaluate(`document.body ? document.body.innerText : ''`, &bodyText),
		chromedp.Evaluate(`document.documentElement ? document.documentElement.outerHTML : ''`, &html),
	)
	if href != "" {
		manifest["href"] = href
	}
	if title != "" {
		manifest["title"] = title
	}
	if bodyText != "" {
		manifest["body_text"] = bodyText
	}
	if data, err := json.MarshalIndent(manifest, "", "  "); err == nil {
		_ = os.WriteFile(basePath+".json", data, 0o644)
	}
	if html != "" {
		_ = os.WriteFile(basePath+".html", []byte(html), 0o644)
	}
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

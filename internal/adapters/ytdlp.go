package adapters

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type YtdlpDownloader struct{}

func NewYtdlpDownloader() *YtdlpDownloader {
	return &YtdlpDownloader{}
}

func (d *YtdlpDownloader) Download(ctx context.Context, url string) (string, string, error) {
	timestamp := time.Now().Unix()
	videoPath := fmt.Sprintf("download_%d.mp4", timestamp)
	audioPath := fmt.Sprintf("audio_%d.wav", timestamp)

	videoCandidates := []downloadStrategy{
		{
			name: "direct mp4",
			match: func(rawURL string) bool {
				return strings.HasSuffix(strings.ToLower(strings.TrimSpace(rawURL)), ".mp4")
			},
			run: func(ctx context.Context, rawURL, targetPath string) error {
				cmd := exec.CommandContext(ctx, "curl", "-k", "-L", rawURL, "-o", targetPath)
				out, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("curl: %w: %s", err, strings.TrimSpace(string(out)))
				}
				return nil
			},
		},
		{
			name: "yt-dlp plain",
			run: func(ctx context.Context, rawURL, targetPath string) error {
				return runYtDLP(ctx, rawURL, targetPath)
			},
		},
		{
			name: "yt-dlp browser headers",
			run: func(ctx context.Context, rawURL, targetPath string) error {
				return runYtDLP(ctx, rawURL, targetPath,
					"--add-header", "Referer:https://www.youtube.com/",
					"--add-header", "Origin:https://www.youtube.com",
					"--user-agent", defaultBrowserUserAgent(),
				)
			},
		},
		{
			name: "yt-dlp web clients",
			run: func(ctx context.Context, rawURL, targetPath string) error {
				return runYtDLP(ctx, rawURL, targetPath,
					"--extractor-args", "youtube:player_client=web,web_creator,mweb,tv,android",
				)
			},
		},
	}

	if browser := strings.TrimSpace(os.Getenv("YTDLP_COOKIES_FROM_BROWSER")); browser != "" {
		videoCandidates = append(videoCandidates, downloadStrategy{
			name: "yt-dlp cookies-from-browser",
			run: func(ctx context.Context, rawURL, targetPath string) error {
				return runYtDLP(ctx, rawURL, targetPath,
					"--cookies-from-browser", browser,
				)
			},
		})
	}

	if impersonate := strings.TrimSpace(os.Getenv("YTDLP_IMPERSONATE")); impersonate != "" {
		videoCandidates = append(videoCandidates, downloadStrategy{
			name: "yt-dlp impersonate",
			run: func(ctx context.Context, rawURL, targetPath string) error {
				return runYtDLP(ctx, rawURL, targetPath,
					"--impersonate", impersonate,
				)
			},
		})
	}

	if jsRuntime := strings.TrimSpace(os.Getenv("YTDLP_JS_RUNTIME")); jsRuntime != "" {
		videoCandidates = append(videoCandidates, downloadStrategy{
			name: "yt-dlp js runtime",
			run: func(ctx context.Context, rawURL, targetPath string) error {
				return runYtDLP(ctx, rawURL, targetPath,
					"--js-runtimes", jsRuntime,
				)
			},
		})
	}

	if path, err := downloadWithFallbacks(ctx, url, videoPath, videoCandidates); err != nil {
		return "", "", err
	} else {
		videoPath = path
	}

	// Extract Audio (using WAV for better compatibility with Whisper and robustness)
	cmdAud := exec.CommandContext(ctx, "ffmpeg",
		"-y", "-i", videoPath,
		"-ar", "16000", "-ac", "1", // Whisper optimal settings
		audioPath,
	)
	if out, err := cmdAud.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("ffmpeg audio extract: %v, output: %s", err, string(out))
	}

	return videoPath, audioPath, nil
}

type downloadStrategy struct {
	name  string
	match func(string) bool
	run   func(context.Context, string, string) error
}

func downloadWithFallbacks(ctx context.Context, rawURL, finalPath string, strategies []downloadStrategy) (string, error) {
	var errs []string
	base := strings.TrimSuffix(finalPath, filepath.Ext(finalPath))

	for idx, strategy := range strategies {
		if strategy.match != nil && !strategy.match(rawURL) {
			continue
		}

		candidatePath := fmt.Sprintf("%s_%d%s", base, idx, filepath.Ext(finalPath))
		_ = os.Remove(candidatePath)

		if err := strategy.run(ctx, rawURL, candidatePath); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", strategy.name, err))
			_ = os.Remove(candidatePath)
			continue
		}

		_ = os.Remove(finalPath)
		if err := os.Rename(candidatePath, finalPath); err != nil {
			_ = os.Remove(candidatePath)
			return "", fmt.Errorf("finalize download: %w", err)
		}
		return finalPath, nil
	}

	if len(errs) == 0 {
		return "", fmt.Errorf("no download strategy matched url %q", rawURL)
	}
	return "", fmt.Errorf("yt-dlp download failed after %d strategies: %s", len(errs), strings.Join(errs, " | "))
}

func runYtDLP(ctx context.Context, rawURL, targetPath string, extraArgs ...string) error {
	args := []string{
		"--no-check-certificate",
		"-f", "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
		"--no-playlist",
		"-o", targetPath,
	}
	args = append(args, extraArgs...)
	args = append(args, rawURL)

	for _, cmdSpec := range ytDLPCommands(args) {
		cmd := exec.CommandContext(ctx, cmdSpec.name, cmdSpec.args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			if isMissingCommandErr(err) {
				continue
			}
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	return fmt.Errorf("yt-dlp executable not found in PATH and python module fallback unavailable")
}

func defaultBrowserUserAgent() string {
	return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"
}

type ytDLPCommandSpec struct {
	name string
	args []string
}

func ytDLPCommands(args []string) []ytDLPCommandSpec {
	specs := []ytDLPCommandSpec{
		{name: "yt-dlp", args: args},
		{name: "python3", args: append([]string{"-m", "yt_dlp"}, args...)},
		{name: "python", args: append([]string{"-m", "yt_dlp"}, args...)},
	}

	return specs
}

func isMissingCommandErr(err error) bool {
	return errors.Is(err, exec.ErrNotFound)
}

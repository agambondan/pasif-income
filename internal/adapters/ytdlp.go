package adapters

import (
	"context"
	"fmt"
	"os/exec"
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
	audioPath := fmt.Sprintf("audio_%d.mp3", timestamp)

	// If direct MP4 link, use curl
	if strings.HasSuffix(url, ".mp4") {
		cmd := exec.CommandContext(ctx, "curl", "-k", "-L", url, "-o", videoPath)
		if err := cmd.Run(); err != nil {
			return "", "", fmt.Errorf("curl: %v", err)
		}
	} else {
		// Download Video (Limit to 1080p) using yt-dlp
		cmdVid := exec.CommandContext(ctx, "yt-dlp",
			"--no-check-certificate",
			"-f", "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
			"--no-playlist",
			"-o", videoPath,
			url,
		)
		if err := cmdVid.Run(); err != nil {
			return "", "", fmt.Errorf("yt-dlp: %v", err)
		}
	}

	// Extract Audio (using WAV for better compatibility with Whisper and robustness)
	audioPath = fmt.Sprintf("audio_%d.wav", timestamp)
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

package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

func TestQualityControlReviewSuccess(t *testing.T) {
	dir := t.TempDir()
	fakeProbe := filepath.Join(dir, "ffprobe")
	script := `#!/bin/sh
cat <<'JSON'
{"streams":[{"codec_type":"video","width":1080,"height":1920}],"format":{"duration":"12.4"}}
JSON
`
	if err := os.WriteFile(fakeProbe, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake ffprobe: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	voicePath := filepath.Join(dir, "voice.mp3")
	videoPath := filepath.Join(dir, "video.mp4")
	if err := os.WriteFile(voicePath, []byte("voice"), 0o644); err != nil {
		t.Fatalf("write voice file: %v", err)
	}
	if err := os.WriteFile(videoPath, []byte("video content"), 0o644); err != nil {
		t.Fatalf("write video file: %v", err)
	}

	qc := NewQualityControlService("")
	report, err := qc.Review(context.Background(), &domain.Story{
		Title:       "Strong Hook Video",
		Niche:       "stoicism",
		Script:      "This is a long enough script to pass the heuristic quality checks and provide a complete, coherent narration with a clear hook.",
		Scenes:      []domain.Scene{{Timestamp: "0-5s", Visual: "calm visual", Text: "hook line"}},
		Voiceover:   voicePath,
		VideoOutput: videoPath,
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	if !report.Passed {
		t.Fatalf("expected qc to pass, got %#v", report)
	}
	if report.Score <= 0 {
		t.Fatalf("expected positive score, got %#v", report)
	}
}

func TestQualityControlReviewFailsWithoutVideo(t *testing.T) {
	qc := NewQualityControlService("")
	report, err := qc.Review(context.Background(), &domain.Story{
		Title:  "Bad",
		Script: "short",
		Scenes: []domain.Scene{},
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	if report.Passed {
		t.Fatalf("expected qc to fail, got %#v", report)
	}
	if len(report.Issues) == 0 {
		t.Fatalf("expected issues, got %#v", report)
	}
}

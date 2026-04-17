package adapters

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

// FFmpegEditor for Clipping (Podcast)
type FFmpegEditor struct{}

func NewFFmpegEditor() *FFmpegEditor {
	return &FFmpegEditor{}
}

func (e *FFmpegEditor) CropAndRender(ctx context.Context, videoPath string, seg domain.ClipSegment, faceX int) (string, error) {
	slug := strings.ReplaceAll(seg.Headline, " ", "_")
	outputPath := fmt.Sprintf("clip_%s.mp4", slug)
	srtPath := fmt.Sprintf("sub_%s.srt", slug)

	if err := e.generateSRT(srtPath, seg.Words); err != nil {
		return "", fmt.Errorf("generate srt: %v", err)
	}
	defer os.Remove(srtPath)

	subStyle := "Alignment=10,FontSize=24,PrimaryColour=&H00FFFF&,Outline=1"
	filters := fmt.Sprintf("crop=ih*9/16:ih:(%d-((ih*9/16)/2)):0,scale=608:1080,subtitles=%s:force_style='%s'", faceX, srtPath, subStyle)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y", "-i", videoPath, "-ss", seg.StartTime, "-to", seg.EndTime, "-vf", filters,
		"-c:v", "libx264", "-preset", "veryfast", "-c:a", "aac", outputPath,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg error: %v, output: %s", err, string(out))
	}
	return outputPath, nil
}

func (e *FFmpegEditor) generateSRT(path string, words []domain.Word) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for i, w := range words {
		start := formatSRTTime(w.Start)
		end := formatSRTTime(w.End)
		fmt.Fprintf(f, "%d\n%s --> %s\n%s\n\n", i+1, start, end, strings.ToUpper(w.Text))
	}
	return nil
}

func formatSRTTime(seconds float64) string {
	t := time.Duration(seconds * float64(time.Second))
	h, m, s, ms := int(t.Hours()), int(t.Minutes())%60, int(t.Seconds())%60, int(t.Milliseconds())%1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

// FFmpegAssembler for Creation (Faceless)
type FFmpegAssembler struct{}

func NewFFmpegAssembler() *FFmpegAssembler {
	return &FFmpegAssembler{}
}

func (a *FFmpegAssembler) Assemble(ctx context.Context, story *domain.Story) (string, error) {
	outputPath := fmt.Sprintf("faceless_%s.mp4", strings.ReplaceAll(story.Title, " ", "_"))
	args := []string{"-y", "-f", "lavfi", "-i", "color=c=black:s=1080x1920:d=60", "-i", story.Voiceover}

	for _, scene := range story.Scenes {
		if scene.ImagePath != "" {
			args = append(args, "-i", scene.ImagePath)
		} else {
			args = append(args, "-f", "lavfi", "-i", "color=c=gray:s=1080x1920:d=1")
		}
	}

	var filterParts []string
	lastOutput := "0:v"
	for i, scene := range story.Scenes {
		inputIdx := i + 2
		timeRange := strings.TrimSuffix(scene.Timestamp, "s")
		parts := strings.Split(timeRange, "-")
		if len(parts) == 2 {
			scaledName := fmt.Sprintf("s%d", i)
			outputName := fmt.Sprintf("v%d", i)
			filterParts = append(filterParts, fmt.Sprintf("[%d:v]scale=1080:1920[%s]", inputIdx, scaledName))
			filterParts = append(filterParts, fmt.Sprintf("[%s][%s]overlay=enable='between(t,%s,%s)'[%s]", 
				lastOutput, scaledName, parts[0], parts[1], outputName))
			lastOutput = outputName
		}
	}

	// Simple captions for fallback
	for i, scene := range story.Scenes {
		cleanText := strings.ReplaceAll(scene.Text, "'", "")
		timeRange := strings.TrimSuffix(scene.Timestamp, "s")
		parts := strings.Split(timeRange, "-")
		if len(parts) == 2 {
			outputName := fmt.Sprintf("vc%d", i)
			filterParts = append(filterParts, fmt.Sprintf("[%s]drawtext=text='%s':fontcolor=yellow:fontsize=80:x=(w-text_w)/2:y=(h-text_h)/1.5:enable='between(t,%s,%s)'[%s]", 
				lastOutput, cleanText, parts[0], parts[1], outputName))
			lastOutput = outputName
		}
	}

	if len(filterParts) > 0 {
		args = append(args, "-filter_complex", strings.Join(filterParts, ";"), "-map", "["+lastOutput+"]")
	} else {
		args = append(args, "-map", "0:v")
	}
	args = append(args, "-map", "1:a", "-shortest", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", outputPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(out)
		if strings.Contains(outputStr, "No such filter: 'drawtext'") {
			log.Println("Warning: No drawtext filter, retrying without captions.")
			return a.assembleWithoutCaptions(ctx, story)
		}
		return "", fmt.Errorf("ffmpeg failed: %v, output: %s", err, outputStr)
	}
	return outputPath, nil
}

func (a *FFmpegAssembler) assembleWithoutCaptions(ctx context.Context, story *domain.Story) (string, error) {
	outputPath := fmt.Sprintf("faceless_%s.mp4", strings.ReplaceAll(story.Title, " ", "_"))
	args := []string{"-y", "-f", "lavfi", "-i", "color=c=black:s=1080x1920:d=60", "-i", story.Voiceover}
	
	for _, scene := range story.Scenes {
		if scene.ImagePath != "" { args = append(args, "-i", scene.ImagePath) } else { args = append(args, "-f", "lavfi", "-i", "color=c=gray:s=1080x1920:d=1") }
	}

	var filterParts []string
	lastOutput := "0:v"
	for i := range story.Scenes {
		inputIdx := i + 2
		timeRange := strings.TrimSuffix(story.Scenes[i].Timestamp, "s")
		parts := strings.Split(timeRange, "-")
		if len(parts) == 2 {
			scaledName, outputName := fmt.Sprintf("s%d", i), fmt.Sprintf("v%d", i)
			filterParts = append(filterParts, fmt.Sprintf("[%d:v]scale=1080:1920[%s]", inputIdx, scaledName))
			filterParts = append(filterParts, fmt.Sprintf("[%s][%s]overlay=enable='between(t,%s,%s)'[%s]", 
				lastOutput, scaledName, parts[0], parts[1], outputName))
			lastOutput = outputName
		}
	}

	if len(filterParts) > 0 {
		args = append(args, "-filter_complex", strings.Join(filterParts, ";"), "-map", "["+lastOutput+"]")
	} else {
		args = append(args, "-map", "0:v")
	}
	args = append(args, "-map", "1:a", "-shortest", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", outputPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil { return "", fmt.Errorf("ffmpeg fallback error: %v, output: %s", err, string(out)) }
	return outputPath, nil
}

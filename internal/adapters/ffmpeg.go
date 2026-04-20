package adapters

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
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
	brand := story.Branding
	avatarInputIdx := -1
	if brand != nil && strings.TrimSpace(brand.AvatarPath) != "" {
		args = append(args, "-i", brand.AvatarPath)
		avatarInputIdx = 2
	}

	for _, scene := range story.Scenes {
		if scene.ImagePath != "" {
			args = append(args, "-i", scene.ImagePath)
		} else {
			args = append(args, "-f", "lavfi", "-i", "color=c=gray:s=1080x1920:d=1")
		}
	}

	var filterParts []string
	lastOutput := "0:v"
	sceneStartIndex := 2
	if avatarInputIdx != -1 {
		sceneStartIndex = 3
	}
	for i, scene := range story.Scenes {
		inputIdx := i + sceneStartIndex
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

	if brand != nil {
		if avatarInputIdx != -1 {
			avatarScaled := "brand_avatar"
			filterParts = append(filterParts, fmt.Sprintf("[%d:v]scale=220:220[%s]", avatarInputIdx, avatarScaled))
			filterParts, lastOutput = appendAvatarOverlay(filterParts, lastOutput, avatarScaled, brand, story.Voiceover)
		}
		filterParts, lastOutput = appendBrandTextOverlays(filterParts, lastOutput, brand, story.Voiceover)
	}

	// Simple captions for fallback
	for i, scene := range story.Scenes {
		cleanText := escapeDrawText(strings.ReplaceAll(scene.Text, "'", ""))
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
	brand := story.Branding
	avatarInputIdx := -1
	if brand != nil && strings.TrimSpace(brand.AvatarPath) != "" {
		args = append(args, "-i", brand.AvatarPath)
		avatarInputIdx = 2
	}

	for _, scene := range story.Scenes {
		if scene.ImagePath != "" {
			args = append(args, "-i", scene.ImagePath)
		} else {
			args = append(args, "-f", "lavfi", "-i", "color=c=gray:s=1080x1920:d=1")
		}
	}

	var filterParts []string
	lastOutput := "0:v"
	sceneStartIndex := 2
	if avatarInputIdx != -1 {
		sceneStartIndex = 3
	}
	for i := range story.Scenes {
		inputIdx := i + sceneStartIndex
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

	if brand != nil {
		if avatarInputIdx != -1 {
			avatarScaled := "brand_avatar"
			filterParts = append(filterParts, fmt.Sprintf("[%d:v]scale=220:220[%s]", avatarInputIdx, avatarScaled))
			filterParts, lastOutput = appendAvatarOverlay(filterParts, lastOutput, avatarScaled, brand, story.Voiceover)
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
		return "", fmt.Errorf("ffmpeg fallback error: %v, output: %s", err, string(out))
	}
	return outputPath, nil
}

func appendAvatarOverlay(filterParts []string, inputLabel string, avatarLabel string, brand *domain.BrandProfile, voiceoverPath string) ([]string, string) {
	if brand == nil || strings.TrimSpace(avatarLabel) == "" {
		return filterParts, inputLabel
	}

	duration := voiceoverDurationSeconds(voiceoverPath, 60)
	overlayEnd := formatDurationFloat(duration)
	outputName := "brand_avatar_out"
	filterParts = append(filterParts, fmt.Sprintf("[%s][%s]overlay=x=36:y=36:enable='between(t,0,%s)'[%s]", inputLabel, avatarLabel, overlayEnd, outputName))
	return filterParts, outputName
}

func appendBrandTextOverlays(filterParts []string, inputLabel string, brand *domain.BrandProfile, voiceoverPath string) ([]string, string) {
	if brand == nil {
		return filterParts, inputLabel
	}

	duration := voiceoverDurationSeconds(voiceoverPath, 60)
	introDuration := minFloat(duration/8, 4)
	if introDuration < 1.5 {
		introDuration = minFloat(duration, 2.5)
	}
	outroDuration := minFloat(duration/8, 4)
	if outroDuration < 1.5 {
		outroDuration = minFloat(duration, 2.5)
	}
	outroStart := duration - outroDuration
	if outroStart < introDuration+1 {
		outroStart = introDuration + 1
	}
	if outroStart < 0 {
		outroStart = 0
	}

	watermark := escapeDrawText(brand.Watermark)
	persona := escapeDrawText(brand.Persona)
	introText := escapeDrawText(brand.IntroText)
	outroText := escapeDrawText(brand.OutroText)

	watermarkOut := "brand_watermark_out"
	filterParts = append(filterParts, fmt.Sprintf("[%s]drawtext=text='%s':fontcolor=white:fontsize=28:box=1:boxcolor=black@0.45:boxborderw=14:x=w-tw-40:y=42:enable='between(t,0,%s)'[%s]",
		inputLabel, watermark, formatDurationFloat(duration), watermarkOut))

	introOut := "brand_intro_out"
	filterParts = append(filterParts, fmt.Sprintf("[%s]drawbox=x=0:y=0:w=iw:h=ih:color=black@0.38:t=fill:enable='between(t,0,%s)'[%s]",
		watermarkOut, formatDurationFloat(introDuration), introOut))

	introTextOut := "brand_intro_text_out"
	filterParts = append(filterParts, fmt.Sprintf("[%s]drawtext=text='%s':fontcolor=white:fontsize=44:box=1:boxcolor=%s@0.35:boxborderw=20:x=(w-text_w)/2:y=(h-text_h)/2-70:enable='between(t,0,%s)'[%s]",
		introOut, introText, normalizeColor(brand.AccentColor), formatDurationFloat(introDuration), introTextOut))

	personaOut := "brand_intro_persona_out"
	filterParts = append(filterParts, fmt.Sprintf("[%s]drawtext=text='%s':fontcolor=white:fontsize=26:box=1:boxcolor=black@0.4:boxborderw=16:x=(w-text_w)/2:y=(h-text_h)/2+10:enable='between(t,0,%s)'[%s]",
		introTextOut, persona, formatDurationFloat(introDuration), personaOut))

	outroBgOut := "brand_outro_bg_out"
	filterParts = append(filterParts, fmt.Sprintf("[%s]drawbox=x=0:y=0:w=iw:h=ih:color=black@0.45:t=fill:enable='between(t,%s,%s)'[%s]",
		personaOut, formatDurationFloat(outroStart), formatDurationFloat(duration), outroBgOut))

	outroTextOut := "brand_outro_text_out"
	filterParts = append(filterParts, fmt.Sprintf("[%s]drawtext=text='%s':fontcolor=white:fontsize=42:box=1:boxcolor=%s@0.35:boxborderw=20:x=(w-text_w)/2:y=(h-text_h)/2-40:enable='between(t,%s,%s)'[%s]",
		outroBgOut, outroText, normalizeColor(brand.AccentColor), formatDurationFloat(outroStart), formatDurationFloat(duration), outroTextOut))

	return filterParts, outroTextOut
}

func voiceoverDurationSeconds(path string, fallback float64) float64 {
	path = strings.TrimSpace(path)
	if path == "" {
		return fallback
	}
	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path).Output()
	if err != nil {
		return fallback
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func formatDurationFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", value), "0"), ".")
}

func escapeDrawText(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "'", "\\'")
	value = strings.ReplaceAll(value, ":", "\\:")
	value = strings.ReplaceAll(value, ",", "\\,")
	value = strings.ReplaceAll(value, "%", "\\%")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

func normalizeColor(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "#3b82f6"
	}
	return strings.TrimPrefix(value, "#")
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

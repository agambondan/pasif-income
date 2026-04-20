package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/google/generative-ai-go/genai"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

type QualityControlService struct {
	apiKey         string
	minScore       int
	autoRegenerate bool
}

func NewQualityControlService(apiKey string) *QualityControlService {
	return &QualityControlService{
		apiKey:         strings.TrimSpace(apiKey),
		minScore:       qcMinScoreFromEnv(),
		autoRegenerate: qcAutoRegenerateFromEnv(),
	}
}

func (s *QualityControlService) Enabled() bool {
	return s != nil
}

func (s *QualityControlService) AutoRegenerateEnabled() bool {
	return s != nil && s.autoRegenerate
}

func (s *QualityControlService) MinScore() int {
	if s == nil || s.minScore <= 0 {
		return 75
	}
	return s.minScore
}

func (s *QualityControlService) Review(ctx context.Context, story *domain.Story) (*domain.QualityControlReport, error) {
	if s == nil {
		return &domain.QualityControlReport{Passed: true, Score: 100, Summary: "qc disabled", ReviewedAt: time.Now().UTC(), Source: "disabled"}, nil
	}
	if story == nil {
		return nil, errors.New("story is required")
	}

	report := &domain.QualityControlReport{
		Score:      100,
		ReviewedAt: time.Now().UTC(),
		Source:     "heuristics",
	}

	s.applyHeuristicChecks(report, story)
	if probe, err := probeVideoFile(ctx, story.VideoOutput); err != nil {
		report.Warnings = append(report.Warnings, err.Error())
		report.Score -= 10
	} else {
		report.Warnings = append(report.Warnings, probe.warnings...)
		report.Score = minInt(report.Score, probe.score)
		report.Source = "heuristics+probe"
	}

	if story.Voiceover != "" {
		if _, err := os.Stat(story.Voiceover); err != nil {
			report.Issues = append(report.Issues, fmt.Sprintf("voiceover missing: %v", err))
			report.Score -= 20
		}
	}

	if story.Branding == nil {
		report.Warnings = append(report.Warnings, "branding profile missing")
		report.Score -= 5
	} else {
		if strings.TrimSpace(story.Branding.Persona) == "" || strings.TrimSpace(story.Branding.Watermark) == "" {
			report.Warnings = append(report.Warnings, "branding text is incomplete")
			report.Score -= 5
		}
		if strings.TrimSpace(story.Branding.AvatarPath) == "" {
			report.Warnings = append(report.Warnings, "branding avatar missing")
			report.Score -= 5
		} else if _, err := os.Stat(story.Branding.AvatarPath); err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("branding avatar missing: %v", err))
			report.Score -= 5
		}
	}

	if s.apiKey != "" || strings.TrimSpace(os.Getenv("GEMINI_ACCESS_TOKEN")) != "" {
		if aiReport, err := s.reviewWithGemini(ctx, story, report); err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("ai reviewer unavailable: %v", err))
		} else if aiReport != nil {
			report = mergeQCReports(report, aiReport)
			report.Source = "heuristics+probe+gemini"
		}
	}

	report.Score = clampScore(report.Score)
	report.Passed = len(report.Issues) == 0 && report.Score >= s.MinScore()
	if report.Passed {
		if report.Summary == "" {
			report.Summary = "quality control passed"
		}
	} else if report.Summary == "" {
		report.Summary = strings.Join(report.Issues, "; ")
	}
	report.Retryable = len(report.Issues) > 0
	report.RegenPrompt = strings.Join(report.Issues, "; ")
	return report, nil
}

func (s *QualityControlService) applyHeuristicChecks(report *domain.QualityControlReport, story *domain.Story) {
	if strings.TrimSpace(story.Title) == "" {
		report.Issues = append(report.Issues, "title is empty")
		report.Score -= 20
	}
	if len(strings.TrimSpace(story.Script)) < 80 {
		report.Issues = append(report.Issues, "script is too short")
		report.Score -= 20
	}
	if len(story.Scenes) == 0 {
		report.Issues = append(report.Issues, "no scenes generated")
		report.Score -= 30
	}

	seenSceneText := map[string]int{}
	for i, scene := range story.Scenes {
		text := strings.TrimSpace(scene.Text)
		visual := strings.TrimSpace(scene.Visual)
		if text == "" {
			report.Issues = append(report.Issues, fmt.Sprintf("scene %d text is empty", i+1))
			report.Score -= 5
		}
		if visual == "" {
			report.Warnings = append(report.Warnings, fmt.Sprintf("scene %d visual prompt is empty", i+1))
			report.Score -= 2
		}
		seenSceneText[strings.ToLower(text)]++
	}

	for text, count := range seenSceneText {
		if text != "" && count > 1 {
			report.Issues = append(report.Issues, "duplicate scene text detected")
			report.Score -= 10
			break
		}
	}

	if story.VideoOutput == "" {
		report.Issues = append(report.Issues, "video output missing")
		report.Score -= 25
		return
	}
	if info, err := os.Stat(story.VideoOutput); err != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("video output missing: %v", err))
		report.Score -= 30
	} else if info.Size() < 100*1024 {
		report.Warnings = append(report.Warnings, "video output is very small")
		report.Score -= 10
	}
}

type qcProbeResult struct {
	score    int
	warnings []string
}

func probeVideoFile(ctx context.Context, path string) (*qcProbeResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("video output path empty")
	}

	if _, err := exec.LookPath("ffprobe"); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-print_format", "json", "-show_streams", "-show_format", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %v: %s", err, strings.TrimSpace(string(out)))
	}

	var payload struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&payload); err != nil {
		return nil, err
	}

	result := &qcProbeResult{score: 100}
	duration, _ := strconv.ParseFloat(strings.TrimSpace(payload.Format.Duration), 64)
	if duration <= 0 {
		result.warnings = append(result.warnings, "ffprobe duration unavailable")
		result.score -= 5
	}
	if duration > 0 && duration < 5 {
		result.warnings = append(result.warnings, "video duration is shorter than 5s")
		result.score -= 15
	}

	for _, stream := range payload.Streams {
		if stream.CodecType != "video" {
			continue
		}
		if stream.Width > 0 && stream.Height > 0 && stream.Height < stream.Width {
			result.warnings = append(result.warnings, "video is not vertical")
			result.score -= 10
		}
		if stream.Width == 0 || stream.Height == 0 {
			result.warnings = append(result.warnings, "video resolution unavailable")
			result.score -= 5
		}
		break
	}

	result.score = clampScore(result.score)
	return result, nil
}

func (s *QualityControlService) reviewWithGemini(ctx context.Context, story *domain.Story, base *domain.QualityControlReport) (*domain.QualityControlReport, error) {
	var opts []option.ClientOption
	accessToken := strings.TrimSpace(os.Getenv("GEMINI_ACCESS_TOKEN"))
	if accessToken != "" {
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
		opts = append(opts, option.WithTokenSource(tokenSource))
	} else if s.apiKey != "" {
		opts = append(opts, option.WithAPIKey(s.apiKey))
	} else {
		return nil, errors.New("gemini credentials unavailable")
	}

	client, err := genai.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-pro")
	model.ResponseMIMEType = "application/json"

	prompt := buildQCReviewPrompt(story, base)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	var jsonText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			jsonText += string(text)
		}
	}

	var ai struct {
		Passed   bool     `json:"passed"`
		Score    int      `json:"score"`
		Summary  string   `json:"summary"`
		Issues   []string `json:"issues"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(jsonText)), &ai); err != nil {
		return nil, err
	}

	report := &domain.QualityControlReport{
		Passed:     ai.Passed,
		Score:      clampScore(ai.Score),
		Summary:    strings.TrimSpace(ai.Summary),
		Issues:     ai.Issues,
		Warnings:   ai.Warnings,
		Retryable:  len(ai.Issues) > 0,
		ReviewedAt: time.Now().UTC(),
		Source:     "gemini",
	}
	return report, nil
}

func buildQCReviewPrompt(story *domain.Story, base *domain.QualityControlReport) string {
	sceneLines := make([]string, 0, len(story.Scenes))
	for _, scene := range story.Scenes {
		sceneLines = append(sceneLines, fmt.Sprintf("- %s | %s | %s", scene.Timestamp, scene.Visual, scene.Text))
	}

	return strings.TrimSpace(fmt.Sprintf(`
You are a strict quality control reviewer for a faceless short-form video pipeline.
Return JSON only with this shape:
{
  "passed": true,
  "score": 0,
  "summary": "short one-line summary",
  "issues": ["problem 1"],
  "warnings": ["warning 1"]
}

Current heuristic result:
- score: %d
- issues: %s
- warnings: %s

Video title: %s
Niche: %s
Script: %s
Scenes:
%s

Evaluate whether the content is good enough to upload. Flag repetition, weak hook, missing visual clarity, or poor flow.
`, base.Score, strings.Join(base.Issues, "; "), strings.Join(base.Warnings, "; "), story.Title, story.Niche, story.Script, strings.Join(sceneLines, "\n")))
}

func mergeQCReports(base *domain.QualityControlReport, ai *domain.QualityControlReport) *domain.QualityControlReport {
	if base == nil {
		return ai
	}
	if ai == nil {
		return base
	}
	merged := *base
	if ai.Score < merged.Score {
		merged.Score = ai.Score
	}
	if ai.Summary != "" {
		merged.Summary = ai.Summary
	}
	merged.Issues = append(merged.Issues, ai.Issues...)
	merged.Warnings = append(merged.Warnings, ai.Warnings...)
	merged.Retryable = merged.Retryable || ai.Retryable
	return &merged
}

func extractJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func qcMinScoreFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("QC_MIN_SCORE"))
	if raw == "" {
		return 75
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 || value > 100 {
		return 75
	}
	return value
}

func qcAutoRegenerateFromEnv() bool {
	raw := strings.TrimSpace(os.Getenv("QC_AUTO_REGENERATE"))
	if raw == "" {
		return true
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		log.Printf("invalid QC_AUTO_REGENERATE=%q, defaulting to true", raw)
		return true
	}
}

func clampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

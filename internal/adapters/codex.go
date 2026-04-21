package adapters

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

const (
	codexDefaultModel  = "gpt-5.4"
	codexFallbackModel = "gpt-5.2-codex"
	codexEndpointPath  = "/codex/responses"
)

type CodexWriter struct {
	authPath string
	client   *http.Client
}

func NewCodexWriter() *CodexWriter {
	home, _ := os.UserHomeDir()
	return &CodexWriter{
		authPath: filepath.Join(home, ".codex", "auth.json"),
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (c *CodexWriter) getAccessToken() (string, error) {
	if token := strings.TrimSpace(os.Getenv("OPENAI_ACCESS_TOKEN")); token != "" {
		return token, nil
	}
	if token := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); token != "" {
		return token, nil
	}
	if c.authPath != "" {
		data, err := os.ReadFile(c.authPath)
		if err != nil {
			return "", fmt.Errorf("read codex auth: %w", err)
		}

		var auth codexAuthFile
		if err := json.Unmarshal(data, &auth); err != nil {
			return "", fmt.Errorf("parse codex auth: %w", err)
		}

		if token := strings.TrimSpace(auth.Tokens.AccessToken); token != "" {
			return token, nil
		}
		if token := strings.TrimSpace(auth.OpenAIAPIKey); token != "" {
			return token, nil
		}
	}

	return "", errors.New("codex credentials unavailable")
}

func (c *CodexWriter) WriteScript(ctx context.Context, niche, topic string) (*domain.Story, error) {
	token, err := c.getAccessToken()
	if err != nil {
		return nil, err
	}

	story, err := c.writeScriptWithModel(ctx, token, codexDefaultModel, niche, topic)
	if err == nil {
		return story, nil
	}

	if !isCodexUnsupportedModelError(err) {
		return nil, err
	}

	return c.writeScriptWithModel(ctx, token, codexFallbackModel, niche, topic)
}

func (c *CodexWriter) Analyze(ctx context.Context, transcript string) ([]domain.ClipSegment, error) {
	token, err := c.getAccessToken()
	if err != nil {
		return nil, err
	}

	segments, err := c.analyzeWithModel(ctx, token, codexDefaultModel, transcript)
	if err == nil {
		return segments, nil
	}

	if !isCodexUnsupportedModelError(err) {
		return nil, err
	}

	return c.analyzeWithModel(ctx, token, codexFallbackModel, transcript)
}

func (c *CodexWriter) writeScriptWithModel(ctx context.Context, token, model, niche, topic string) (*domain.Story, error) {
	apiURL := "https://chatgpt.com/backend-api" + codexEndpointPath
	instructions := `Act as a Professional Faceless Channel Content Creator.

Rules:
1. NO MUSIC.
2. NO WOMEN.
3. STRICT ISLAMIC SHARIA PRINCIPLES.

Return JSON only.`

	userPrompt := fmt.Sprintf(`Niche: %s
Topic: %s

Create a viral script for a Short video (30-60s).
Output MUST be a JSON object:
{
  "title": "Viral Title",
  "script": "The full spoken narration text...",
  "scenes": [{"timestamp": "0-5s", "visual_prompt": "shot of...", "scene_text": "text"}]
}`, niche, topic)

	body := map[string]any{
		"model":        model,
		"instructions": instructions,
		"stream":       true,
		"store":        false,
		"input": []any{
			map[string]any{
				"role":    "user",
				"content": userPrompt,
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal codex request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("OpenAI-Beta", "responses=v1")
	req.Header.Set("User-Agent", "pasif-income-codex-writer/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		bodyText := strings.TrimSpace(string(bodyBytes))
		if model == codexDefaultModel && isCodexUnsupportedModelBody(resp.StatusCode, bodyText) {
			return nil, &codexUnsupportedModelError{status: resp.StatusCode, body: bodyText}
		}
		return nil, fmt.Errorf("codex api error (%d): %s", resp.StatusCode, bodyText)
	}

	jsonText, err := readCodexStream(resp.Body)
	if err != nil {
		return nil, err
	}

	var story domain.Story
	if err := json.Unmarshal([]byte(extractJSONPayload(jsonText)), &story); err != nil {
		return nil, fmt.Errorf("decode story: %w", err)
	}
	return &story, nil
}

func (c *CodexWriter) analyzeWithModel(ctx context.Context, token, model, transcript string) ([]domain.ClipSegment, error) {
	apiURL := "https://chatgpt.com/backend-api" + codexEndpointPath
	instructions := `Act as a Viral Marketing Expert for podcast clips.

Rules:
1. Do NOT select segments that contain music.
2. Do NOT select segments featuring women.
3. STRICT ISLAMIC SHARIA PRINCIPLES apply.

Return JSON only.`

	userPrompt := fmt.Sprintf(`Transcript:
%s

Find the most engaging short-form clip segments from the transcript.
Output MUST be a JSON array of objects:
[
  {
    "start_time": "120",
    "end_time": "165",
    "headline": "Why AI is the future",
    "viral_score": 95,
    "reasoning": "Strong hook"
  }
]`, transcript)

	body := map[string]any{
		"model":        model,
		"instructions": instructions,
		"stream":       true,
		"store":        false,
		"input": []any{
			map[string]any{
				"role":    "user",
				"content": userPrompt,
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal codex request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("OpenAI-Beta", "responses=v1")
	req.Header.Set("User-Agent", "pasif-income-codex-strategist/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		bodyText := strings.TrimSpace(string(bodyBytes))
		if model == codexDefaultModel && isCodexUnsupportedModelBody(resp.StatusCode, bodyText) {
			return nil, &codexUnsupportedModelError{status: resp.StatusCode, body: bodyText}
		}
		return nil, fmt.Errorf("codex api error (%d): %s", resp.StatusCode, bodyText)
	}

	jsonText, err := readCodexStream(resp.Body)
	if err != nil {
		return nil, err
	}

	var segments []domain.ClipSegment
	if err := json.Unmarshal([]byte(extractJSONPayload(jsonText)), &segments); err != nil {
		return nil, fmt.Errorf("decode clip segments: %w", err)
	}
	return segments, nil
}

type codexUnsupportedModelError struct {
	status int
	body   string
}

func (e *codexUnsupportedModelError) Error() string {
	return fmt.Sprintf("codex model unsupported (%d): %s", e.status, e.body)
}

func isCodexUnsupportedModelError(err error) bool {
	var target *codexUnsupportedModelError
	return errors.As(err, &target)
}

func isCodexUnsupportedModelBody(status int, body string) bool {
	if status != http.StatusBadRequest && status != http.StatusForbidden {
		return false
	}
	lowerBody := strings.ToLower(body)
	return strings.Contains(lowerBody, "not supported when using codex with a chatgpt account") ||
		strings.Contains(lowerBody, "gpt-5.4") && strings.Contains(lowerBody, "not supported")
}

func readCodexStream(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var text strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			if data == "[DONE]" {
				break
			}
			continue
		}

		var event codexSSEEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "response.output_text.delta":
			if event.Delta != "" {
				text.WriteString(event.Delta)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("codex stream read error: %w", err)
	}

	output := strings.TrimSpace(text.String())
	if output == "" {
		return "", errors.New("empty codex response")
	}
	return output, nil
}

type codexAPIResponse struct {
	ID     string      `json:"id"`
	Object string      `json:"object"`
	Model  string      `json:"model"`
	Status string      `json:"status"`
	Output []codexItem `json:"output"`
}

type codexItem struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Role      string         `json:"role,omitempty"`
	Phase     string         `json:"phase,omitempty"`
	Content   []codexContent `json:"content,omitempty"`
	CallID    string         `json:"call_id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Arguments string         `json:"arguments,omitempty"`
	Summary   []codexSummary `json:"summary,omitempty"`
}

type codexContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexSummary struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexSSEEvent struct {
	Type     string            `json:"type"`
	Delta    string            `json:"delta,omitempty"`
	ItemID   string            `json:"item_id,omitempty"`
	Item     *codexItem        `json:"item,omitempty"`
	Response *codexAPIResponse `json:"response,omitempty"`
}

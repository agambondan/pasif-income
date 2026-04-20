package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/google/generative-ai-go/genai"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

const geminiDefaultModel = "gemini-2.5-pro"

type GeminiAgent struct {
	apiKey string
}

func NewGeminiAgent(key string) *GeminiAgent {
	return &GeminiAgent{apiKey: strings.TrimSpace(key)}
}

type GeminiWriter struct {
	apiKey string
}

func NewGeminiWriter(key string) *GeminiWriter {
	return &GeminiWriter{apiKey: strings.TrimSpace(key)}
}

type GeminiCommentResponder struct {
	apiKey string
}

func NewGeminiCommentResponder(key string) *GeminiCommentResponder {
	return &GeminiCommentResponder{apiKey: strings.TrimSpace(key)}
}

func geminiClientOptions(apiKey string) ([]option.ClientOption, error) {
	var opts []option.ClientOption

	if accessToken := strings.TrimSpace(os.Getenv("GEMINI_ACCESS_TOKEN")); accessToken != "" {
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
		opts = append(opts, option.WithTokenSource(tokenSource))
		return opts, nil
	}

	if accessToken := strings.TrimSpace(os.Getenv("GEMINI_ACCESS_TOKEN_FROM_OAUTH_CREDS")); accessToken != "" {
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
		opts = append(opts, option.WithTokenSource(tokenSource))
		return opts, nil
	}

	if accessToken := readGeminiOAuthToken(); accessToken != "" {
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
		opts = append(opts, option.WithTokenSource(tokenSource))
		return opts, nil
	}

	if token, ok := geminiTokenFromJSON(strings.TrimSpace(apiKey)); ok {
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		opts = append(opts, option.WithTokenSource(tokenSource))
		return opts, nil
	}

	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}

	return opts, nil
}

func geminiTokenFromJSON(raw string) (string, bool) {
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return "", false
	}

	var parsed struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", false
	}

	token := strings.TrimSpace(parsed.Token)
	if token == "" {
		return "", false
	}
	return token, true
}

func newGeminiClient(ctx context.Context, apiKey string) (*genai.Client, error) {
	opts, err := geminiClientOptions(apiKey)
	if err != nil {
		return nil, err
	}
	if len(opts) == 0 {
		return nil, errors.New("gemini credentials unavailable")
	}
	return genai.NewClient(ctx, opts...)
}

func geminiResponseText(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0] == nil || resp.Candidates[0].Content == nil {
		return "", errors.New("empty gemini response")
	}

	var text strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		switch v := part.(type) {
		case genai.Text:
			text.WriteString(string(v))
		}
	}

	output := strings.TrimSpace(text.String())
	if output == "" {
		return "", errors.New("empty gemini response text")
	}
	return output, nil
}

func (g *GeminiAgent) Analyze(ctx context.Context, transcript string) ([]domain.ClipSegment, error) {
	client, err := newGeminiClient(ctx, g.apiKey)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel(geminiDefaultModel)
	model.ResponseMIMEType = "application/json"

	prompt := `Act as a Viral Marketing Expert. Analyze the following transcript and identify the most engaging segments for short-form clips.

STRICT RULES (MUST FOLLOW):
1. Do NOT select any segments that contain music.
2. Do NOT select any segments featuring women.
3. The selected content MUST strictly adhere to Islamic sharia principles.

Output MUST be a JSON array of objects: [{"start_time": "120", "end_time": "165", "headline": "Why AI is the future", "viral_score": 95, "reasoning": "Strong hook"}]
Transcript: ` + transcript

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	jsonText, err := geminiResponseText(resp)
	if err != nil {
		return nil, err
	}

	var segments []domain.ClipSegment
	if err := json.Unmarshal([]byte(extractJSONPayload(jsonText)), &segments); err != nil {
		return nil, fmt.Errorf("decode clip segments: %w", err)
	}
	return segments, nil
}

func (g *GeminiWriter) WriteScript(ctx context.Context, niche, topic string) (*domain.Story, error) {
	client, err := newGeminiClient(ctx, g.apiKey)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel(geminiDefaultModel)
	model.ResponseMIMEType = "application/json"

	prompt := `Act as a Professional Faceless Channel Content Creator.
Niche: ` + niche + `
Topic: ` + topic + `

STRICT RULES:
1. NO MUSIC.
2. NO WOMEN.
3. STRICT ISLAMIC SHARIA PRINCIPLES.

Create a viral script for a Short video (30-60s).
Output MUST be a JSON object:
{
  "title": "Viral Title",
  "script": "The full spoken narration text...",
  "scenes": [{"timestamp": "0-5s", "visual_prompt": "shot of...", "scene_text": "text"}]
}
`

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	jsonText, err := geminiResponseText(resp)
	if err != nil {
		return nil, err
	}

	var story domain.Story
	if err := json.Unmarshal([]byte(extractJSONPayload(jsonText)), &story); err != nil {
		return nil, fmt.Errorf("decode story: %w", err)
	}
	return &story, nil
}

func (g *GeminiCommentResponder) DraftReply(ctx context.Context, niche, topic, videoTitle, commentText, persona string) (string, error) {
	client, err := newGeminiClient(ctx, g.apiKey)
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(geminiDefaultModel)
	model.ResponseMIMEType = "application/json"

	prompt := fmt.Sprintf(
		`Act as a community manager for a faceless YouTube channel.

Brand persona: %s
Niche: %s
Topic: %s
Video title: %s

Viewer comment:
%s

Rules:
1. Write a short, warm, human reply.
2. Match the language of the viewer comment when possible.
3. Do not mention being AI.
4. Do not add hashtags or affiliate links.
5. Keep it under 2 sentences unless the comment asks for clarification.

Return JSON only in this shape:
{"reply":"..."}`,
		persona, niche, topic, videoTitle, commentText,
	)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	jsonText, err := geminiResponseText(resp)
	if err != nil {
		return "", err
	}

	var payload struct {
		Reply string `json:"reply"`
	}
	if err := json.Unmarshal([]byte(extractJSONPayload(jsonText)), &payload); err != nil {
		return "", fmt.Errorf("decode reply: %w", err)
	}

	reply := strings.TrimSpace(payload.Reply)
	if reply == "" {
		return "", fmt.Errorf("empty community reply")
	}
	return reply, nil
}

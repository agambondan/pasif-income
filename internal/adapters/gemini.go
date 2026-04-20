package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/google/generative-ai-go/genai"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

type GeminiAgent struct {
	apiKey string
}

func NewGeminiAgent(key string) *GeminiAgent {
	return &GeminiAgent{apiKey: key}
}

func geminiClientOptions(apiKey string) []option.ClientOption {
	var opts []option.ClientOption
	accessToken := os.Getenv("GEMINI_ACCESS_TOKEN")

	if accessToken != "" {
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
		opts = append(opts, option.WithTokenSource(tokenSource))
	} else if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return opts
}

// Analyze for Clipping (Podcast)
func (g *GeminiAgent) Analyze(ctx context.Context, transcript string) ([]domain.ClipSegment, error) {
	client, err := genai.NewClient(ctx, geminiClientOptions(g.apiKey)...)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-pro")
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

	var jsonStr string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			jsonStr = string(text)
		}
	}

	var segments []domain.ClipSegment
	err = json.Unmarshal([]byte(jsonStr), &segments)
	return segments, err
}

// GeminiWriter for Creation (Faceless)
type GeminiWriter struct {
	apiKey string
}

func NewGeminiWriter(key string) *GeminiWriter {
	return &GeminiWriter{apiKey: key}
}

func (g *GeminiWriter) WriteScript(ctx context.Context, niche, topic string) (*domain.Story, error) {
	client, err := genai.NewClient(ctx, geminiClientOptions(g.apiKey)...)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-pro")
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

	var jsonStr string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			jsonStr = string(text)
		}
	}

	var story domain.Story
	err = json.Unmarshal([]byte(jsonStr), &story)
	return &story, err
}

type GeminiCommentResponder struct {
	apiKey string
}

func NewGeminiCommentResponder(key string) *GeminiCommentResponder {
	return &GeminiCommentResponder{apiKey: key}
}

func (g *GeminiCommentResponder) DraftReply(ctx context.Context, niche, topic, videoTitle, commentText, persona string) (string, error) {
	client, err := genai.NewClient(ctx, geminiClientOptions(g.apiKey)...)
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-pro")
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

	var jsonStr string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			jsonStr = string(text)
		}
	}

	var payload struct {
		Reply string `json:"reply"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
		return "", err
	}

	reply := strings.TrimSpace(payload.Reply)
	if reply == "" {
		return "", fmt.Errorf("empty community reply")
	}
	return reply, nil
}

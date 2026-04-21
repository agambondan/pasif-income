package services

import (
	"context"
	"errors"
	"github.com/agambondan/pasif-income/internal/core/domain"
	"testing"
)

// Fakes
type fakeScriptWriter struct {
	err error
}

func (m *fakeScriptWriter) WriteScript(ctx context.Context, niche string, topic string) (*domain.Story, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &domain.Story{
		Title:  "Test Story",
		Script: "Test Script",
		Scenes: []domain.Scene{{Visual: "Test Visual"}},
	}, nil
}

type fakeVoiceGenerator struct {
	err error
}

func (m *fakeVoiceGenerator) GenerateVO(ctx context.Context, text string, voiceType string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "voice.mp3", nil
}

type fakeImageGenerator struct {
	err error
}

func (m *fakeImageGenerator) GenerateImage(ctx context.Context, prompt string, sceneID int) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "image.png", nil
}

type fakeVideoAssembler struct {
	err error
}

func (m *fakeVideoAssembler) Assemble(ctx context.Context, story *domain.Story) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "video.mp4", nil
}

type fakeUploader struct {
	err error
}

func (m *fakeUploader) Upload(ctx context.Context, filePath string, title string, description string) error {
	return m.err
}

func TestGenerateContent_Success(t *testing.T) {
	s := NewGeneratorService(
		&fakeScriptWriter{},
		nil,
		&fakeVoiceGenerator{},
		&fakeImageGenerator{},
		&fakeVideoAssembler{},
		&fakeUploader{},
		nil,
		nil,
		nil,
	)

	story, err := s.GenerateContent(context.Background(), "motivation", "discipline", "en-US-Standard-A")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if story == nil || story.VideoOutput != "video.mp4" {
		t.Fatalf("expected story with video path, got %#v", story)
	}
}

func TestGenerateContent_ScriptError(t *testing.T) {
	expectedErr := errors.New("failed to write script")
	s := NewGeneratorService(
		&fakeScriptWriter{err: expectedErr},
		nil,
		&fakeVoiceGenerator{},
		&fakeImageGenerator{},
		&fakeVideoAssembler{},
		&fakeUploader{},
		nil,
		nil,
		nil,
	)

	_, err := s.GenerateContent(context.Background(), "motivation", "discipline", "en-US-Standard-A")
	if err == nil || err.Error() != "script writer (Gemini & Codex): failed to write script" {
		t.Errorf("expected script writer error, got %v", err)
	}
}

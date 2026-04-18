package services

import (
	"context"
	"errors"
	"github.com/agambondan/pasif-income/internal/core/domain"
	"testing"
)

// Mocks
type mockScriptWriter struct {
	err error
}

func (m *mockScriptWriter) WriteScript(ctx context.Context, niche string, topic string) (*domain.Story, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &domain.Story{
		Title:  "Test Story",
		Script: "Test Script",
		Scenes: []domain.Scene{{Visual: "Test Visual"}},
	}, nil
}

type mockVoiceGenerator struct {
	err error
}

func (m *mockVoiceGenerator) GenerateVO(ctx context.Context, text string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "voice.mp3", nil
}

type mockImageGenerator struct {
	err error
}

func (m *mockImageGenerator) GenerateImage(ctx context.Context, prompt string, sceneID int) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "image.png", nil
}

type mockVideoAssembler struct {
	err error
}

func (m *mockVideoAssembler) Assemble(ctx context.Context, story *domain.Story) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "video.mp4", nil
}

type mockUploader struct {
	err error
}

func (m *mockUploader) Upload(ctx context.Context, filePath string, title string, description string) error {
	return m.err
}

func TestGenerateContent_Success(t *testing.T) {
	s := NewGeneratorService(
		&mockScriptWriter{},
		&mockVoiceGenerator{},
		&mockImageGenerator{},
		&mockVideoAssembler{},
		&mockUploader{},
	)

	story, err := s.GenerateContent(context.Background(), "motivation", "discipline")
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
		&mockScriptWriter{err: expectedErr},
		&mockVoiceGenerator{},
		&mockImageGenerator{},
		&mockVideoAssembler{},
		&mockUploader{},
	)

	_, err := s.GenerateContent(context.Background(), "motivation", "discipline")
	if err == nil || err.Error() != "script writer: failed to write script" {
		t.Errorf("expected script writer error, got %v", err)
	}
}

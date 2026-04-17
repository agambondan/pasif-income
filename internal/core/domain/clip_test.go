package domain

import (
	"testing"
)

func TestClipCreation(t *testing.T) {
	clip := Clip{
		ID:        "test-id",
		SourceID:  "source-id",
		Headline:  "Test Headline",
		StartTime: "00:00:01",
		EndTime:   "00:00:10",
		Status:    "processing",
	}

	if clip.ID != "test-id" {
		t.Errorf("expected ID test-id, got %s", clip.ID)
	}

	if clip.Headline != "Test Headline" {
		t.Errorf("expected Headline 'Test Headline', got %s", clip.Headline)
	}
}

func TestWordCreation(t *testing.T) {
	word := Word{
		Text:  "hello",
		Start: 1.0,
		End:   1.5,
	}

	if word.Text != "hello" {
		t.Errorf("expected Text hello, got %s", word.Text)
	}

	if word.Start != 1.0 {
		t.Errorf("expected Start 1.0, got %f", word.Start)
	}
}

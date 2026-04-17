package adapters

import (
	"context"
	"log"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

type MockTranscriber struct{}

func NewMockTranscriber() *MockTranscriber {
	return &MockTranscriber{}
}

func (m *MockTranscriber) Transcribe(ctx context.Context, audioPath string) (string, []domain.Word, error) {
	log.Println("Using MOCK Transcriber (No Whisper API needed)")

	// Mock text that might be found in a podcast
	text := "Artificial intelligence is changing the way we create content forever. It's not just about speed, it's about the quality and creativity that AI brings to the table."

	// Mock words with timestamps (first 10 seconds)
	words := []domain.Word{
		{Text: "Artificial", Start: 0.1, End: 0.5},
		{Text: "intelligence", Start: 0.6, End: 1.2},
		{Text: "is", Start: 1.3, End: 1.4},
		{Text: "changing", Start: 1.5, End: 2.0},
		{Text: "the", Start: 2.1, End: 2.2},
		{Text: "way", Start: 2.3, End: 2.5},
		{Text: "we", Start: 2.6, End: 2.7},
		{Text: "create", Start: 2.8, End: 3.2},
		{Text: "content", Start: 3.3, End: 3.8},
		{Text: "forever.", Start: 3.9, End: 4.5},
	}

	return text, words, nil
}

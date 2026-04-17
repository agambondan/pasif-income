package adapters

import (
	"context"
	"github.com/agambondan/pasif-income/internal/core/domain"
	"log"
)

type MockStrategist struct{}

func NewMockStrategist() *MockStrategist {
	return &MockStrategist{}
}

func (m *MockStrategist) Analyze(ctx context.Context, transcript string) ([]domain.ClipSegment, error) {
	log.Println("Using MOCK Strategist (No Gemini API needed)")
	
	return []domain.ClipSegment{
		{
			StartTime: "1",
			EndTime:   "6",
			Headline:  "Short AI Discussion",
			Score:     95,
			Reasoning: "Capturing a 5 second clip",
		},
	}, nil
}

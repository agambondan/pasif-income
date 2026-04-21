package adapters

import (
	"context"
	"fmt"
	"log"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
)

type FallbackStrategist struct {
	primary  ports.StrategistAgent
	fallback ports.StrategistAgent
}

func NewFallbackStrategist(primary ports.StrategistAgent, fallback ports.StrategistAgent) *FallbackStrategist {
	return &FallbackStrategist{
		primary:  primary,
		fallback: fallback,
	}
}

func (s *FallbackStrategist) Analyze(ctx context.Context, transcript string) ([]domain.ClipSegment, error) {
	if s != nil && s.primary != nil {
		segments, err := s.primary.Analyze(ctx, transcript)
		if err == nil {
			return segments, nil
		}
		log.Printf("primary strategist failed: %v; trying fallback", err)
		if s.fallback == nil {
			return nil, err
		}
	}

	if s == nil || s.fallback == nil {
		return nil, fmt.Errorf("strategist unavailable")
	}

	return s.fallback.Analyze(ctx, transcript)
}

package services

import (
	"testing"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

func TestSynthesizeIdeasUsesSignals(t *testing.T) {
	signals := []domain.TrendSignal{
		{Query: "how to build discipline", Source: "google-trends", Score: 95, Context: "high intent"},
		{Query: "mindset reset", Source: "autocomplete-yt", Score: 80, Context: "autocomplete"},
	}

	ideas := synthesizeIdeas("stoicism", signals, 2)
	if len(ideas) != 2 {
		t.Fatalf("expected 2 ideas, got %d", len(ideas))
	}
	if ideas[0].Title == "" || ideas[0].Hook == "" || ideas[0].Angle == "" {
		t.Fatalf("expected populated idea, got %#v", ideas[0])
	}
	if ideas[0].SearchQuery != signals[0].Query {
		t.Fatalf("expected first idea to follow first signal, got %#v", ideas[0])
	}
}

func TestFallbackTrendSignalsProduceIdeas(t *testing.T) {
	fallback := fallbackTrendSignals("fitness")
	if len(fallback) == 0 {
		t.Fatal("expected fallback signals")
	}
	ideas := synthesizeIdeas("fitness", fallback, 3)
	if len(ideas) != 3 {
		t.Fatalf("expected 3 ideas, got %d", len(ideas))
	}
	if ideas[0].TrendSource == "" {
		t.Fatalf("expected source on idea, got %#v", ideas[0])
	}
}

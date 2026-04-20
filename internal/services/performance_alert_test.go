package services

import (
	"testing"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

func TestPerformanceAlertService_AssessDetectsSharpDrop(t *testing.T) {
	svc := &PerformanceAlertService{
		dropThreshold: 30,
		minBaseline:   500,
	}

	history := []domain.VideoMetricSnapshot{
		{
			ExternalID:  "video-1",
			Platform:    "youtube",
			AccountID:   "acct-1",
			Niche:       "stoicism",
			VideoTitle:  "Calm under pressure",
			ViewCount:   600,
			CollectedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ExternalID:  "video-1",
			Platform:    "youtube",
			AccountID:   "acct-1",
			Niche:       "stoicism",
			VideoTitle:  "Calm under pressure",
			ViewCount:   320,
			CollectedAt: time.Now(),
		},
	}

	alerts := svc.Assess(history)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Level != "high" {
		t.Fatalf("expected high alert, got %#v", alerts[0])
	}
	if alerts[0].Metric != "views" {
		t.Fatalf("expected view alert, got %#v", alerts[0])
	}
}

func TestFailoverAccountsForDestinationPrefersSameAuthMethod(t *testing.T) {
	current := domain.ConnectedAccount{
		ID:          "acct-1",
		UserID:      99,
		PlatformID:  "youtube",
		AuthMethod:  domain.AuthMethodAPI,
		DisplayName: "Primary",
	}
	accounts := []domain.ConnectedAccount{
		current,
		{
			ID:          "acct-2",
			UserID:      99,
			PlatformID:  "youtube",
			AuthMethod:  domain.AuthMethodChromiumProfile,
			DisplayName: "Browser backup",
			CreatedAt:   time.Now().Add(-time.Hour),
		},
		{
			ID:          "acct-3",
			UserID:      99,
			PlatformID:  "youtube",
			AuthMethod:  domain.AuthMethodAPI,
			DisplayName: "API backup",
			CreatedAt:   time.Now(),
		},
		{
			ID:         "acct-4",
			UserID:     99,
			PlatformID: "tiktok",
			AuthMethod: domain.AuthMethodChromiumProfile,
		},
	}

	candidates := failoverAccountsForDestination(accounts, current)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].ID != "acct-3" {
		t.Fatalf("expected same auth method backup first, got %#v", candidates[0])
	}
}

package services

import (
	"context"
	"testing"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

func TestAgentServiceLogPersistsEvent(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewAgentService(repo)

	svc.Log(context.Background(), "job-123", domain.AgentEventThought, "analyze hook", map[string]any{"score": 92})

	if len(repo.agentEvents) != 1 {
		t.Fatalf("expected 1 stored event, got %d", len(repo.agentEvents))
	}
	event := repo.agentEvents[0]
	if event.JobID != "job-123" {
		t.Fatalf("expected job id job-123, got %#v", event)
	}
	if event.Type != domain.AgentEventThought {
		t.Fatalf("expected thought event, got %#v", event)
	}
	if event.Content == "" {
		t.Fatalf("expected content to be populated, got %#v", event)
	}
	if event.Timestamp.IsZero() {
		t.Fatalf("expected timestamp to be populated, got %#v", event.Timestamp)
	}
	if got := event.Metadata["score"]; got != 92 {
		t.Fatalf("expected metadata score 92, got %#v", got)
	}
}

func TestAgentServiceSubscribeBroadcastsEvent(t *testing.T) {
	svc := NewAgentService(&fakeRepository{})

	ch, cleanup := svc.Subscribe("job-456")
	defer cleanup()

	svc.Log(context.Background(), "job-456", domain.AgentEventResult, "segment selected", nil)

	select {
	case event := <-ch:
		if event.JobID != "job-456" {
			t.Fatalf("expected job-456, got %#v", event)
		}
		if event.Type != domain.AgentEventResult {
			t.Fatalf("expected result event, got %#v", event)
		}
		if event.Content != "segment selected" {
			t.Fatalf("unexpected content: %#v", event.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for broadcast event")
	}
}

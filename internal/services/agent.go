package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
)

type AgentService struct {
	repo ports.Repository
	
	// Channels for real-time streaming
	mu          sync.RWMutex
	subscribers map[string][]chan domain.AgentEvent // jobID -> channels
}

func NewAgentService(repo ports.Repository) *AgentService {
	return &AgentService{
		repo:        repo,
		subscribers: make(map[string][]chan domain.AgentEvent),
	}
}

func (s *AgentService) Log(ctx context.Context, jobID string, eventType domain.AgentEventType, content string, metadata map[string]any) {
	event := domain.AgentEvent{
		ID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		JobID:     jobID,
		Type:      eventType,
		Content:   content,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	// Persist
	if s.repo != nil {
		_ = s.repo.SaveAgentEvent(ctx, &event)
	}

	// Broadcast
	s.mu.RLock()
	subs, ok := s.subscribers[jobID]
	s.mu.RUnlock()

	if ok {
		for _, ch := range subs {
			select {
			case ch <- event:
			default:
				// Drop if channel is full
			}
		}
	}
}

func (s *AgentService) Subscribe(jobID string) (chan domain.AgentEvent, func()) {
	ch := make(chan domain.AgentEvent, 100)
	
	s.mu.Lock()
	s.subscribers[jobID] = append(s.subscribers[jobID], ch)
	s.mu.Unlock()

	cleanup := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		
		subs := s.subscribers[jobID]
		for i, sub := range subs {
			if sub == ch {
				s.subscribers[jobID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
	}

	return ch, cleanup
}

func (s *AgentService) ListEvents(ctx context.Context, jobID string) ([]domain.AgentEvent, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.ListAgentEvents(ctx, jobID)
}

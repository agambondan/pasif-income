package domain

import "time"

type AgentEventType string

const (
	AgentEventThought AgentEventType = "thought"
	AgentEventAction  AgentEventType = "action"
	AgentEventResult  AgentEventType = "result"
	AgentEventSystem  AgentEventType = "system"
)

type AgentEvent struct {
	ID        string           `json:"id"`
	JobID     string           `json:"job_id"`
	Type      AgentEventType   `json:"type"`
	Content   string           `json:"content"`
	Metadata  map[string]any   `json:"metadata,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
}

type AgentSession struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

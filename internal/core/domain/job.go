package domain

import "time"

// GenerationJob represents a persisted generation request and its status.
type GenerationJob struct {
	ID          string     `json:"id"`
	Niche       string     `json:"niche"`
	Topic       string     `json:"topic"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

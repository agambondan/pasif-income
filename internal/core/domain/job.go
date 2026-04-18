package domain

import "time"

type Destination struct {
	Platform  string `json:"platform"`
	AccountID string `json:"account_id"`
}

type DistributionJob struct {
	ID              int       `json:"id"`
	GenerationJobID string    `json:"generation_job_id"`
	AccountID       string    `json:"account_id"`
	Platform        string    `json:"platform"`
	Status          string    `json:"status"`
	StatusDetail    string    `json:"status_detail"`
	ExternalID      string    `json:"external_id"`
	Error           string    `json:"error"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GenerationJob represents a persisted generation request and its status.
type GenerationJob struct {
	ID          string     `json:"id"`
	Niche       string     `json:"niche"`
	Topic       string     `json:"topic"`
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	VideoPath   string     `json:"video_path,omitempty"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

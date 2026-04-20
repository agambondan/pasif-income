package domain

import "time"

type Destination struct {
	Platform  string `json:"platform"`
	AccountID string `json:"account_id"`
}

type DistributionJob struct {
	ID              int        `json:"id"`
	GenerationJobID string     `json:"generation_job_id"`
	AccountID       string     `json:"account_id"`
	Platform        string     `json:"platform"`
	Status          string     `json:"status"`
	StatusDetail    string     `json:"status_detail"`
	ExternalID      string     `json:"external_id"`
	Error           string     `json:"error"`
	ScheduledAt     *time.Time `json:"scheduled_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// GenerationJob represents a persisted generation request and its status.
type GenerationJob struct {
	ID          string     `json:"id"`
	Niche       string     `json:"niche"`
	Topic       string     `json:"topic"`
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	PinComment  string     `json:"pin_comment,omitempty"`
	VideoPath   string     `json:"video_path,omitempty"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type VideoMetricSnapshot struct {
	ID                int       `json:"id"`
	UserID            int       `json:"user_id"`
	GenerationJobID   string    `json:"generation_job_id"`
	DistributionJobID int       `json:"distribution_job_id"`
	AccountID         string    `json:"account_id"`
	Platform          string    `json:"platform"`
	Niche             string    `json:"niche"`
	ExternalID        string    `json:"external_id"`
	VideoTitle        string    `json:"video_title"`
	ViewCount         uint64    `json:"view_count"`
	LikeCount         uint64    `json:"like_count"`
	CommentCount      uint64    `json:"comment_count"`
	CollectedAt       time.Time `json:"collected_at"`
}

package domain

import "time"

type PerformanceAlert struct {
	ID            string    `json:"id"`
	Level         string    `json:"level"`
	Platform      string    `json:"platform"`
	AccountID     string    `json:"account_id"`
	Niche         string    `json:"niche"`
	ExternalID    string    `json:"external_id"`
	VideoTitle    string    `json:"video_title"`
	Metric        string    `json:"metric"`
	CurrentValue  uint64    `json:"current_value"`
	PreviousValue uint64    `json:"previous_value"`
	DropPercent   float64   `json:"drop_percent"`
	Message       string    `json:"message"`
	CreatedAt     time.Time `json:"created_at"`
}

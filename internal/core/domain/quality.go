package domain

import "time"

type QualityControlReport struct {
	Passed      bool      `json:"passed"`
	Score       int       `json:"score"`
	Summary     string    `json:"summary"`
	Issues      []string  `json:"issues"`
	Warnings    []string  `json:"warnings"`
	Retryable   bool      `json:"retryable"`
	ReviewedAt  time.Time `json:"reviewed_at"`
	RegenPrompt string    `json:"regen_prompt,omitempty"`
	Source      string    `json:"source,omitempty"`
}

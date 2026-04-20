package domain

import "time"

type CommunityReplyDraft struct {
	ID                int        `json:"id"`
	UserID            int        `json:"user_id"`
	GenerationJobID   string     `json:"generation_job_id"`
	DistributionJobID int        `json:"distribution_job_id"`
	AccountID         string     `json:"account_id"`
	Platform          string     `json:"platform"`
	Niche             string     `json:"niche"`
	VideoTitle        string     `json:"video_title"`
	ExternalCommentID string     `json:"external_comment_id"`
	ParentCommentID   string     `json:"parent_comment_id"`
	CommentAuthor     string     `json:"comment_author"`
	CommentText       string     `json:"comment_text"`
	SuggestedReply    string     `json:"suggested_reply"`
	Status            string     `json:"status"`
	PostedExternalID  string     `json:"posted_external_id,omitempty"`
	RepliedAt         *time.Time `json:"replied_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

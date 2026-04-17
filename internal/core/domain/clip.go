package domain

// Word represents a single word with its timing
type Word struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// ClipSegment represents a viral moment identified by AI
type ClipSegment struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Headline  string `json:"headline"`
	Score     int    `json:"viral_score"`
	Reasoning string `json:"reasoning"`
	Words     []Word `json:"words,omitempty"` // Captured words for captions
}

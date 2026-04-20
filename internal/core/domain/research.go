package domain

import "time"

type TrendSignal struct {
	Query   string `json:"query"`
	Source  string `json:"source"`
	Score   int    `json:"score"`
	Link    string `json:"link,omitempty"`
	Context string `json:"context,omitempty"`
}

type IdeaSuggestion struct {
	Title       string `json:"title"`
	Hook        string `json:"hook"`
	Angle       string `json:"angle"`
	SearchQuery string `json:"search_query"`
	TrendSource string `json:"trend_source"`
	Score       int    `json:"score"`
	Reason      string `json:"reason"`
}

type TrendResearchResult struct {
	Niche       string           `json:"niche"`
	Seed        string           `json:"seed"`
	Signals     []TrendSignal    `json:"signals"`
	Ideas       []IdeaSuggestion `json:"ideas"`
	Warnings    []string         `json:"warnings,omitempty"`
	CollectedAt time.Time        `json:"collected_at"`
}

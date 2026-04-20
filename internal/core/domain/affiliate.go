package domain

type AffiliateProduct struct {
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	CTA        string   `json:"cta"`
	Disclosure string   `json:"disclosure"`
	PinComment string   `json:"pin_comment"`
	Tags       []string `json:"tags,omitempty"`
}

type AffiliatePlan struct {
	Product     AffiliateProduct `json:"product"`
	MatchSource string           `json:"match_source"`
	Score       int              `json:"score"`
	Description string           `json:"description"`
	PinComment  string           `json:"pin_comment"`
}

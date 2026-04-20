package domain

type BrandProfile struct {
	Persona     string `json:"persona"`
	Watermark   string `json:"watermark"`
	IntroText   string `json:"intro_text"`
	OutroText   string `json:"outro_text"`
	AvatarPath  string `json:"avatar_path"`
	AccentColor string `json:"accent_color,omitempty"`
	AssetKey    string `json:"asset_key,omitempty"`
	Description string `json:"description,omitempty"`
}

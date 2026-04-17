package domain

type Story struct {
	Title       string   `json:"title"`
	Niche       string   `json:"niche"`
	Script      string   `json:"script"`
	Scenes      []Scene  `json:"scenes"`
	Voiceover   string   `json:"voiceover_path"`
	VideoOutput string   `json:"video_path"`
}

type Scene struct {
	Timestamp   string `json:"timestamp"`
	Visual      string `json:"visual_prompt"`
	Text        string `json:"scene_text"`
	ImagePath   string `json:"image_path"`
}

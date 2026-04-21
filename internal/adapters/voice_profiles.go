package adapters

import "strings"

type VoiceProfile struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Language string `json:"language"`
	TLD      string `json:"tld"`
}

var defaultVoiceProfile = VoiceProfile{
	ID:       "en-US-Standard-A",
	Label:    "English (US)",
	Language: "en",
	TLD:      "com",
}

var supportedVoiceProfiles = []VoiceProfile{
	defaultVoiceProfile,
	{
		ID:       "en-GB-Standard-A",
		Label:    "English (UK)",
		Language: "en",
		TLD:      "co.uk",
	},
	{
		ID:       "en-AU-Standard-A",
		Label:    "English (Australia)",
		Language: "en",
		TLD:      "com.au",
	},
	{
		ID:       "en-IN-Standard-A",
		Label:    "English (India)",
		Language: "en",
		TLD:      "co.in",
	},
	{
		ID:       "id-ID-Standard-A",
		Label:    "Bahasa Indonesia",
		Language: "id",
		TLD:      "com",
	},
	{
		ID:       "es-ES-Standard-A",
		Label:    "Español (ES)",
		Language: "es",
		TLD:      "com",
	},
	{
		ID:       "pt-BR-Standard-A",
		Label:    "Português (BR)",
		Language: "pt",
		TLD:      "com.br",
	},
	{
		ID:       "fr-FR-Standard-A",
		Label:    "Français (FR)",
		Language: "fr",
		TLD:      "com",
	},
}

func SupportedVoiceProfiles() []VoiceProfile {
	out := make([]VoiceProfile, len(supportedVoiceProfiles))
	copy(out, supportedVoiceProfiles)
	return out
}

func ResolveVoiceProfile(voiceType string) (VoiceProfile, bool) {
	candidate := strings.TrimSpace(voiceType)
	if candidate == "" {
		return defaultVoiceProfile, true
	}
	for _, profile := range supportedVoiceProfiles {
		if strings.EqualFold(profile.ID, candidate) {
			return profile, true
		}
	}
	return defaultVoiceProfile, false
}

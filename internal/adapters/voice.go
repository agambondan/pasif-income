package adapters

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type VoiceAdapter struct {
	defaultVoiceType string
}

func NewVoiceAdapter(voiceType string) *VoiceAdapter {
	if strings.TrimSpace(voiceType) == "" {
		voiceType = defaultVoiceProfile.ID
	}
	return &VoiceAdapter{defaultVoiceType: voiceType}
}

func (v *VoiceAdapter) GenerateVO(ctx context.Context, text string, voiceType string) (string, error) {
	outputPath := "output_vo.mp3"

	profile, supported := ResolveVoiceProfile(voiceType)
	if !supported {
		profile, _ = ResolveVoiceProfile(v.defaultVoiceType)
	}

	cmd := exec.CommandContext(
		ctx,
		"gtts-cli",
		text,
		"--output",
		outputPath,
		"--lang",
		profile.Language,
		"--tld",
		profile.TLD,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return "", fmt.Errorf("gtts-cli failed: %w: %s", err, detail)
		}
		return "", fmt.Errorf("gtts-cli failed: %w", err)
	}

	return outputPath, nil
}

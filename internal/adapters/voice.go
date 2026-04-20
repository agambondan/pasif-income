package adapters

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type VoiceAdapter struct {
	voiceType string
}

func NewVoiceAdapter(voiceType string) *VoiceAdapter {
	return &VoiceAdapter{voiceType: voiceType}
}

func (v *VoiceAdapter) GenerateVO(ctx context.Context, text string) (string, error) {
	outputPath := "output_vo.mp3"
	cmd := exec.CommandContext(ctx, "gtts-cli", text, "--output", outputPath)
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

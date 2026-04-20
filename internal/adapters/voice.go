package adapters

import (
	"context"
	"fmt"
	"os/exec"
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
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gtts-cli failed: %w", err)
	}

	return outputPath, nil
}

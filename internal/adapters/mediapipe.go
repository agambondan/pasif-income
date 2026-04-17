package adapters

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

type PythonVisionAgent struct {
	scriptPath string
}

func NewPythonVisionAgent(script string) *PythonVisionAgent {
	return &PythonVisionAgent{scriptPath: script}
}

func (v *PythonVisionAgent) DetectFaceCenter(ctx context.Context, videoPath, startTime, endTime string) (int, error) {
	cmd := exec.CommandContext(ctx, "python3", v.scriptPath, videoPath, startTime, endTime)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	result := strings.TrimSpace(string(out))
	xCoord, err := strconv.Atoi(result)
	if err != nil {
		return 0, err
	}

	return xCoord, nil
}

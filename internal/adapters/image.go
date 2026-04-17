package adapters

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
)

type StableDiffusionAdapter struct {
	apiURL string
}

func NewStableDiffusionAdapter(url string) *StableDiffusionAdapter {
	if url == "" {
		url = "http://localhost:7860/sdapi/v1/txt2img"
	}
	return &StableDiffusionAdapter{apiURL: url}
}

func (a *StableDiffusionAdapter) GenerateImage(ctx context.Context, prompt string, sceneID int) (string, error) {
	outputPath := fmt.Sprintf("scene_%d.png", sceneID)
	log.Printf("Real Image Gen [SD] scene %d: %s\n", sceneID, prompt)

	payload := map[string]interface{}{
		"prompt":            prompt + ", high quality, 4k, cinematic, realistic",
		"negative_prompt":   "low quality, blurry, distorted, text, watermark",
		"steps":             20,
		"width":             1080,
		"height":            1920, // Vertical for Shorts/TikTok
		"cfg_scale":         7.5,
		"sampler_name":      "Euler a",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// Fallback to placeholder if SD is not running
		log.Printf("Stable Diffusion not reachable, falling back to placeholder.\n")
		return a.fallbackPlaceholder(ctx, sceneID)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sd api error: %d", resp.StatusCode)
	}

	var result struct {
		Images []string `json:"images"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Images) == 0 {
		return "", fmt.Errorf("no image returned from sd")
	}

	// Decode base64
	imgData, err := base64.StdEncoding.DecodeString(result.Images[0])
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(outputPath, imgData, 0644)
	return outputPath, err
}

func (a *StableDiffusionAdapter) fallbackPlaceholder(ctx context.Context, sceneID int) (string, error) {
	outputPath := fmt.Sprintf("scene_%d.png", sceneID)
	exec.CommandContext(ctx, "ffmpeg", "-y", "-f", "lavfi", "-i", "color=c=random:s=1080x1920:d=1", "-vframes", "1", outputPath).Run()
	return outputPath, nil
}

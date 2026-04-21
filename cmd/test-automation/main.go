package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/agambondan/pasif-income/internal/adapters"
)

func main() {
	fmt.Println("--- content-factory Automation Stress Test ---")

	// 1. Setup paths
	cwd, _ := os.Getwd()
	videoPath := filepath.Join(cwd, "dummy_video.mp4")
	
	// Gunakan profil YouTube lo yang beneran
	profilePath := filepath.Join(cwd, "chromium-profiles", "youtube", "agam_pro234_at_gmail_com")

	// 2. Initialize Runner
	runner := adapters.NewChromiumRunnerFromEnv()
	
	// Set Headless ke FALSE biar keliatan di layar
	os.Setenv("BROWSER_HEADLESS", "false")
	os.Setenv("BROWSER_AUTOMATION_TIMEOUT_SECONDS", "300") // 5 menit max

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	fmt.Printf("Testing upload for video: %s\n", videoPath)
	fmt.Printf("Using profile: %s\n", profilePath)

	progress := func(stage string) {
		fmt.Printf("[PROG] Current Stage: %s\n", stage)
	}

	// 3. Trigger Automation
	// Note: automation ini bakal hit URL YouTube Upload
	url := "https://www.youtube.com/upload"
	
	fmt.Println("Launching browser in 3... 2... 1...")
	err := runner.AutomateUpload(ctx, profilePath, url, videoPath, "Dummy Automation Test", "This is an automated test from content-factory v1.2", "youtube", progress)

	if err != nil {
		log.Fatalf("FATAL: Automation failed: %v\n", err)
	}

	fmt.Println("SUCCESS! Automation finished without crashing.")
}

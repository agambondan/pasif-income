package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/agambondan/pasif-income/internal/adapters"
	"github.com/agambondan/pasif-income/internal/core/ports"
	"github.com/agambondan/pasif-income/internal/services"
)

func main() {
	log.Println("--- Faceless Content Generator Starting ---")

	// Get API Key from env
	apiKey := os.Getenv("GEMINI_API_KEY")
	accessToken := os.Getenv("GEMINI_ACCESS_TOKEN")
	if apiKey == "" && accessToken == "" {
		log.Fatal("ERROR: GEMINI_API_KEY or GEMINI_ACCESS_TOKEN not set.")
	}

	// 1. Initialize Adapters
	writer := adapters.NewGeminiWriter(apiKey)
	voice := adapters.NewVoiceAdapter("en-US-Standard-A")
	image := adapters.NewStableDiffusionAdapter(os.Getenv("SD_API_URL"))
	assembler := adapters.NewFFmpegAssembler()
	uploader, err := newUploaderFromEnv()
	if err != nil {
		log.Fatalf("Uploader init failed: %v", err)
	}

	// 2. Initialize Service
	service := services.NewGeneratorService(writer, voice, image, assembler, uploader, services.NewQualityControlService(apiKey))

	// 3. Execution
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	niche := os.Getenv("NICHE")
	if niche == "" {
		niche = "stoicism"
	}
	topic := os.Getenv("TOPIC")
	if topic == "" {
		topic = "how to control your mind"
	}

	log.Printf("Starting generator for Niche: %s, Topic: %s\n", niche, topic)
	story, err := service.GenerateContent(ctx, niche, topic)
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	log.Printf("Generated video at: %s\n", story.VideoOutput)
	log.Println("Generation completed successfully!")
}

func newUploaderFromEnv() (ports.Uploader, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9002"
	}
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "admin"
	}
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "secretpassword"
	}
	bucket := os.Getenv("MINIO_BUCKET")
	if bucket == "" {
		bucket = "clips"
	}

	uploader, err := adapters.NewMinIOUploader(endpoint, accessKey, secretKey, bucket, "YouTube Shorts")
	if err != nil {
		return nil, fmt.Errorf("init minio uploader: %w", err)
	}

	return uploader, nil
}

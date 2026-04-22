package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/adapters"
	"github.com/agambondan/pasif-income/internal/core/ports"
	"github.com/agambondan/pasif-income/internal/services"
)

func main() {
	log.Println("--- Faceless Content Generator Starting ---")

	voiceTypeFlag := flag.String("voice-type", "", "voice preset id for gTTS, e.g. en-US-Standard-A")
	listVoiceTypesFlag := flag.Bool("list-voice-types", false, "list supported voice presets and exit")
	flag.Parse()

	if *listVoiceTypesFlag {
		printSupportedVoiceTypes()
		return
	}

	// Get API Key from env
	apiKey := os.Getenv("GEMINI_API_KEY")
	if !adapters.HasGeminiCredentials() && !adapters.HasCodexCredentials() {
		log.Fatal("ERROR: no Gemini or Codex credentials available.")
	}

	// 1. Initialize Adapters
	writer := adapters.NewGeminiWriter(apiKey)
	codexWriter := adapters.NewCodexWriter()
	voiceType := resolveVoiceType(*voiceTypeFlag)
	voice := adapters.NewVoiceAdapter(voiceType)
	image := adapters.NewStableDiffusionAdapter(os.Getenv("SD_API_URL"))
	assembler := adapters.NewFFmpegAssembler()
	uploader, err := newUploaderFromEnv()
	if err != nil {
		log.Fatalf("Uploader init failed: %v", err)
	}

	// 2. Initialize Service
	service := services.NewGeneratorService(nil, writer, codexWriter, voice, image, assembler, uploader, services.NewBrandingService(image), services.NewAffiliateService(), services.NewQualityControlService(apiKey), nil)

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
	story, err := service.GenerateContent(ctx, "", niche, topic, voiceType)
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

func defaultVoiceTypeFromEnv() string {
	if voiceType := strings.TrimSpace(os.Getenv("VOICE_TYPE")); voiceType != "" {
		return voiceType
	}
	return "en-US-Standard-A"
}

func resolveVoiceType(flagValue string) string {
	if voiceType := strings.TrimSpace(flagValue); voiceType != "" {
		if profile, ok := adapters.ResolveVoiceProfile(voiceType); ok {
			return profile.ID
		}
		log.Printf("Warning: unsupported voice type %q, falling back to default", voiceType)
		return defaultVoiceTypeFromEnv()
	}
	return defaultVoiceTypeFromEnv()
}

func printSupportedVoiceTypes() {
	profiles := adapters.SupportedVoiceProfiles()
	fmt.Println("Supported voice presets:")
	for _, profile := range profiles {
		fmt.Printf("- %s | %s | lang=%s | tld=%s\n", profile.ID, profile.Label, profile.Language, profile.TLD)
	}
	fmt.Printf("Default: %s\n", defaultVoiceTypeFromEnv())
}

package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
)

type GeneratorService struct {
	writer    ports.ScriptWriter
	voice     ports.VoiceGenerator
	image     ports.ImageGenerator
	assembler ports.VideoAssembler
	uploader  ports.Uploader
	branding  *BrandingService
	quality   *QualityControlService
}

func NewGeneratorService(w ports.ScriptWriter, v ports.VoiceGenerator, i ports.ImageGenerator, a ports.VideoAssembler, u ports.Uploader, branding *BrandingService, qc *QualityControlService) *GeneratorService {
	return &GeneratorService{
		writer:    w,
		voice:     v,
		image:     i,
		assembler: a,
		uploader:  u,
		branding:  branding,
		quality:   qc,
	}
}

func (s *GeneratorService) GenerateContent(ctx context.Context, niche string, topic string) (*domain.Story, error) {
	log.Printf("Starting content generation for Niche: %s, Topic: %s\n", niche, topic)

	attemptTopic := topic
	maxAttempts := 1
	if s.quality != nil && s.quality.AutoRegenerateEnabled() {
		maxAttempts = 2
	}

	var cleanup func()
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			log.Printf("QC retry attempt %d for topic %q\n", attempt+1, attemptTopic)
		}

		story, attemptCleanup, err := s.generateAttempt(ctx, niche, attemptTopic)
		if err != nil {
			if attemptCleanup != nil {
				attemptCleanup()
			}
			return nil, err
		}
		if cleanup != nil {
			cleanup()
		}
		cleanup = attemptCleanup

		if s.quality != nil {
			report, err := s.quality.Review(ctx, story)
			if err != nil {
				if cleanup != nil {
					cleanup()
					cleanup = nil
				}
				return nil, fmt.Errorf("quality control: %w", err)
			}
			log.Printf("QC report: passed=%t score=%d summary=%s\n", report.Passed, report.Score, report.Summary)
			if !report.Passed {
				if attempt+1 < maxAttempts {
					attemptTopic = fmt.Sprintf("%s [qc revision: %s]", topic, strings.TrimSpace(report.RegenPrompt))
					continue
				}
				if cleanup != nil {
					cleanup()
					cleanup = nil
				}
				return nil, fmt.Errorf("quality control failed: %s", strings.Join(report.Issues, "; "))
			}
		}

		description := fmt.Sprintf("#%s #%s #ai #faceless", niche, strings.ReplaceAll(topic, " ", ""))
		if err := s.uploader.Upload(ctx, story.VideoOutput, story.Title, description); err != nil {
			log.Printf("Warning: Upload failed: %v", err)
		}
		return story, nil
	}

	return nil, fmt.Errorf("quality control failed after retries")
}

func (s *GeneratorService) generateAttempt(ctx context.Context, niche string, topic string) (*domain.Story, func(), error) {
	// 1. Write Script & Scene Plan
	story, err := s.writer.WriteScript(ctx, niche, topic)
	if err != nil {
		return nil, nil, fmt.Errorf("script writer: %v", err)
	}
	log.Printf("Script generated: %s\n", story.Title)

	// 2. Generate Voiceover
	voPath, err := s.voice.GenerateVO(ctx, story.Script)
	if err != nil {
		return nil, nil, fmt.Errorf("voice generator: %v", err)
	}
	story.Voiceover = voPath
	log.Printf("Voiceover generated: %s\n", voPath)

	// 3. Generate Images for each Scene
	log.Println("3. Generating images for each scene...")
	var tempFiles []string
	tempFiles = append(tempFiles, voPath)

	for idx, scene := range story.Scenes {
		imgPath, err := s.image.GenerateImage(ctx, scene.Visual, idx)
		if err != nil {
			log.Printf("Warning: Image generation for scene %d failed: %v", idx, err)
			continue
		}
		story.Scenes[idx].ImagePath = imgPath
		tempFiles = append(tempFiles, imgPath)
	}

	if s.branding != nil {
		brand, err := s.branding.Resolve(ctx, niche)
		if err != nil {
			log.Printf("Warning: branding resolve failed: %v", err)
		}
		story.Branding = brand
	}

	// 4. Assemble Video
	videoPath, err := s.assembler.Assemble(ctx, story)
	if err != nil {
		return nil, nil, fmt.Errorf("video assembler: %v", err)
	}
	story.VideoOutput = videoPath
	log.Printf("Final video assembled: %s\n", videoPath)

	cleanup := func() {
		log.Println("Cleanup: Removing temporary image and audio files...")
		for _, file := range tempFiles {
			_ = os.Remove(file)
		}
		_ = os.Remove(videoPath)
	}

	return story, cleanup, nil
}

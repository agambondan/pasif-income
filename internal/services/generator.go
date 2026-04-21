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
	repo        ports.Repository
	writer      ports.ScriptWriter
	codexWriter ports.ScriptWriter
	voice       ports.VoiceGenerator
	image       ports.ImageGenerator
	assembler   ports.VideoAssembler
	uploader    ports.Uploader
	branding    *BrandingService
	affiliate   *AffiliateService
	quality     *QualityControlService
}

func NewGeneratorService(repo ports.Repository, w ports.ScriptWriter, cw ports.ScriptWriter, v ports.VoiceGenerator, i ports.ImageGenerator, a ports.VideoAssembler, u ports.Uploader, branding *BrandingService, affiliate *AffiliateService, qc *QualityControlService) *GeneratorService {
	return &GeneratorService{
		repo:        repo,
		writer:      w,
		codexWriter: cw,
		voice:       v,
		image:       i,
		assembler:   a,
		uploader:    u,
		branding:    branding,
		affiliate:   affiliate,
		quality:     qc,
	}
}

func (s *GeneratorService) GenerateContent(ctx context.Context, jobID string, niche string, topic string, voiceType string) (*domain.Story, error) {
	log.Printf("Starting content generation for Job: %s, Niche: %s, Topic: %s\n", jobID, niche, topic)

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

		story, attemptCleanup, err := s.generateAttempt(ctx, jobID, niche, attemptTopic, voiceType)
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
			s.updateProgress(ctx, jobID, "quality_control", 90)
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

		description := story.Description
		if strings.TrimSpace(description) == "" {
			description = fmt.Sprintf("#%s #%s #ai #faceless", niche, strings.ReplaceAll(topic, " ", ""))
		}
		
		s.updateProgress(ctx, jobID, "uploading", 95)
		if err := s.uploader.Upload(ctx, story.VideoOutput, story.Title, description); err != nil {
			log.Printf("Warning: Upload failed: %v", err)
		}
		
		s.updateProgress(ctx, jobID, "completed", 100)
		return story, nil
	}

	return nil, fmt.Errorf("quality control failed after retries")
}

func (s *GeneratorService) updateProgress(ctx context.Context, jobID, stage string, pct int) {
	if s.repo != nil {
		_ = s.repo.UpdateJobProgress(ctx, jobID, stage, pct)
	}
}

func (s *GeneratorService) generateAttempt(ctx context.Context, jobID string, niche string, topic string, voiceType string) (*domain.Story, func(), error) {
	// 1. Write Script & Scene Plan - with Fallback
	s.updateProgress(ctx, jobID, "writing_script", 15)
	story, err := s.writer.WriteScript(ctx, niche, topic)
	if err != nil {
		log.Printf("Gemini failed: %v. Attempting Codex fallback...\n", err)
		if s.codexWriter != nil {
			story, err = s.codexWriter.WriteScript(ctx, niche, topic)
		}

		if err != nil {
			return nil, nil, fmt.Errorf("script writer (Gemini & Codex): %v", err)
		}
	}
	log.Printf("Script generated: %s\n", story.Title)
	s.updateProgress(ctx, jobID, "writing_script", 30)

	// 2. Generate Voiceover
	s.updateProgress(ctx, jobID, "generating_audio", 45)
	voPath, err := s.voice.GenerateVO(ctx, story.Script, voiceType)
	if err != nil {
		return nil, nil, fmt.Errorf("voice generator: %v", err)
	}
	story.Voiceover = voPath
	log.Printf("Voiceover generated: %s\n", voPath)
	s.updateProgress(ctx, jobID, "generating_audio", 55)

	// 3. Generate Images for each Scene
	s.updateProgress(ctx, jobID, "generating_visuals", 65)
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
	s.updateProgress(ctx, jobID, "generating_visuals", 75)

	if s.branding != nil {
		brand, err := s.branding.Resolve(ctx, niche)
		if err != nil {
			log.Printf("Warning: branding resolve failed: %v", err)
		}
		story.Branding = brand
	}

	if s.affiliate != nil {
		plan := s.affiliate.Build(niche, topic)
		if plan != nil {
			story.Affiliate = plan
			story.Description = plan.Description
			story.PinComment = plan.PinComment
		}
	}
	if strings.TrimSpace(story.Description) == "" {
		story.Description = fmt.Sprintf("#%s #%s #ai #faceless", niche, strings.ReplaceAll(topic, " ", ""))
	}

	// 4. Assemble Video
	s.updateProgress(ctx, jobID, "assembling_video", 85)
	videoPath, err := s.assembler.Assemble(ctx, story)
	if err != nil {
		return nil, nil, fmt.Errorf("video assembler: %v", err)
	}
	story.VideoOutput = videoPath
	log.Printf("Final video assembled: %s\n", videoPath)
	s.updateProgress(ctx, jobID, "assembling_video", 90)

	cleanup := func() {
		log.Println("Cleanup: Removing temporary image and audio files...")
		for _, file := range tempFiles {
			_ = os.Remove(file)
		}
		_ = os.Remove(videoPath)
	}

	return story, cleanup, nil
}

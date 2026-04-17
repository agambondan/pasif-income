package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/agambondan/pasif-income/internal/core/ports"
)

type GeneratorService struct {
	writer    ports.ScriptWriter
	voice     ports.VoiceGenerator
	image     ports.ImageGenerator
	assembler ports.VideoAssembler
	uploader  ports.Uploader
}

func NewGeneratorService(w ports.ScriptWriter, v ports.VoiceGenerator, i ports.ImageGenerator, a ports.VideoAssembler, u ports.Uploader) *GeneratorService {
	return &GeneratorService{w, v, i, a, u}
}

func (s *GeneratorService) GenerateContent(ctx context.Context, niche string, topic string) error {
	log.Printf("Starting content generation for Niche: %s, Topic: %s\n", niche, topic)

	// 1. Write Script & Scene Plan
	story, err := s.writer.WriteScript(ctx, niche, topic)
	if err != nil {
		return fmt.Errorf("script writer: %v", err)
	}
	log.Printf("Script generated: %s\n", story.Title)

	// 2. Generate Voiceover
	voPath, err := s.voice.GenerateVO(ctx, story.Script)
	if err != nil {
		return fmt.Errorf("voice generator: %v", err)
	}
	story.Voiceover = voPath
	log.Printf("Voiceover generated: %s\n", voPath)

	// 3. Generate Images for each Scene
	log.Println("3. Generating images for each scene...")
	var tempFiles []string
	tempFiles = append(tempFiles, voPath)

	for idx, scene := range story.Scenes {
		img_path, err := s.image.GenerateImage(ctx, scene.Visual, idx)
		if err != nil {
			log.Printf("Warning: Image generation for scene %d failed: %v", idx, err)
			continue
		}
		story.Scenes[idx].ImagePath = img_path
		tempFiles = append(tempFiles, img_path)
	}

	// 4. Assemble Video
	videoPath, err := s.assembler.Assemble(ctx, story)
	if err != nil {
		return fmt.Errorf("video assembler: %v", err)
	}
	story.VideoOutput = videoPath
	log.Printf("Final video assembled: %s\n", videoPath)

	// 5. Upload to Platform
	description := fmt.Sprintf("#%s #%s #ai #faceless", niche, strings.ReplaceAll(topic, " ", ""))
	err = s.uploader.Upload(ctx, videoPath, story.Title, description)
	if err != nil {
		log.Printf("Warning: Upload failed: %v", err)
	}

	// 6. Cleanup Temporary Files
	log.Println("Cleanup: Removing temporary image and audio files...")
	for _, file := range tempFiles {
		os.Remove(file)
	}

	return nil
}

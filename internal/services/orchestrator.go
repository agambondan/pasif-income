package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
)

type WorkflowService struct {
	downloader  ports.Downloader
	transcriber ports.Transcriber
	agent       ports.StrategistAgent
	editor      ports.VideoEditor
	vision      ports.VisionAgent
	storage     ports.Storage
	repo        ports.Repository
	agentLog    *AgentService
}

func NewWorkflowService(
	d ports.Downloader,
	t ports.Transcriber,
	a ports.StrategistAgent,
	e ports.VideoEditor,
	v ports.VisionAgent,
	s ports.Storage,
	r ports.Repository,
	al *AgentService,
) *WorkflowService {
	return &WorkflowService{d, t, a, e, v, s, r, al}
}

func (s *WorkflowService) logThought(ctx context.Context, jobID, content string) {
	if s.agentLog != nil {
		s.agentLog.Log(ctx, jobID, domain.AgentEventThought, content, nil)
	}
}

func (s *WorkflowService) logAction(ctx context.Context, jobID, content string, meta map[string]any) {
	if s.agentLog != nil {
		s.agentLog.Log(ctx, jobID, domain.AgentEventAction, content, meta)
	}
}

func (s *WorkflowService) logResult(ctx context.Context, jobID, content string) {
	if s.agentLog != nil {
		s.agentLog.Log(ctx, jobID, domain.AgentEventResult, content, nil)
	}
}

func (s *WorkflowService) RunPipeline(ctx context.Context, videoURL string) error {
	jobID := "clipper_" + time.Now().Format("20060102150405")
	s.logThought(ctx, jobID, fmt.Sprintf("1. Downloading video: %s", videoURL))
	s.logAction(ctx, jobID, "downloader.Download", map[string]any{"url": videoURL})

	videoPath, audioPath, err := s.downloader.Download(ctx, videoURL)
	if err != nil {
		return fmt.Errorf("downloader: %v", err)
	}
	s.logResult(ctx, jobID, "Download complete.")

	s.logThought(ctx, jobID, "2. Transcribing audio...")
	s.logAction(ctx, jobID, "transcriber.Transcribe", nil)
	transcriptText, allWords, err := s.transcriber.Transcribe(ctx, audioPath)
	if err != nil {
		return fmt.Errorf("transcriber: %v", err)
	}
	s.logResult(ctx, jobID, "Transcription complete.")

	s.logThought(ctx, jobID, "3. AI analyzing transcript for viral hooks...")
	s.logAction(ctx, jobID, "strategist.Analyze", nil)
	segments, err := s.agent.Analyze(ctx, transcriptText)
	if err != nil {
		return fmt.Errorf("strategist: %v", err)
	}

	log.Printf("Found %d viral segments. Starting production...\n", len(segments))
	s.logResult(ctx, jobID, fmt.Sprintf("Found %d viral segments.", len(segments)))

	for i, seg := range segments {
		log.Printf("4. Processing clip %d: %s (%s - %s)\n", i+1, seg.Headline, seg.StartTime, seg.EndTime)
		s.logThought(ctx, jobID, fmt.Sprintf("Processing clip %d: %s", i+1, seg.Headline))

		// Filter words for this specific segment
		seg.Words = s.filterWords(allWords, seg.StartTime, seg.EndTime)

		// Vision: Find face position
		s.logAction(ctx, jobID, "vision.DetectFaceCenter", map[string]any{"clip": seg.Headline})
		faceX, err := s.vision.DetectFaceCenter(ctx, videoPath, seg.StartTime, seg.EndTime)
		if err != nil {
			log.Printf("Warning: Vision failed, using center crop. Err: %v", err)
			faceX = 1920 / 2 // Default center for 1080p video
		}

		// Editor: Crop 9:16 and Render with captions
		s.logAction(ctx, jobID, "editor.CropAndRender", nil)
		outputPath, err := s.editor.CropAndRender(ctx, videoPath, seg, faceX)
		if err != nil {
			return fmt.Errorf("failed to render clip %d: %v", i+1, err)
		}

		// 5. Save to Storage (MinIO)
		s.logAction(ctx, jobID, "storage.Upload", nil)
		objectName := fmt.Sprintf("%d_%s.mp4", time.Now().Unix(), strings.ReplaceAll(seg.Headline, " ", "_"))
		remoteURL, err := s.storage.Upload(ctx, outputPath, objectName)
		if err != nil {
			log.Printf("Warning: Upload to storage failed: %v", err)
		} else {
			log.Printf("Clip uploaded to: %s\n", remoteURL)
		}

		// 6. Save Metadata to DB (Postgres)
		err = s.repo.SaveClip(ctx, &seg, videoURL, remoteURL)
		if err != nil {
			log.Printf("Warning: Database save failed: %v", err)
		}

		log.Printf("Success! Clip processed: %s\n", outputPath)
		s.logResult(ctx, jobID, fmt.Sprintf("Clip %d processed successfully.", i+1))
	}

	return nil
}

func (s *WorkflowService) filterWords(allWords []domain.Word, startStr, endStr string) []domain.Word {
	// Parse strings to float (assumes seconds)
	var start, end float64
	fmt.Sscanf(startStr, "%f", &start)
	fmt.Sscanf(endStr, "%f", &end)

	var filtered []domain.Word
	for _, w := range allWords {
		if w.Start >= start && w.End <= end {
			// Offset timing to start from 0 for the clip
			w.Start -= start
			w.End -= start
			filtered = append(filtered, w)
		}
	}
	return filtered
}

package ports

import (
	"context"
	"github.com/agambondan/pasif-income/internal/core/domain"
)

// Shared Ports
type Storage interface {
	Upload(ctx context.Context, filePath string, objectName string) (url string, err error)
}

type Repository interface {
	SaveClip(ctx context.Context, clip *domain.ClipSegment, videoURL string) error
	UpdateStatus(ctx context.Context, clipID string, status string) error
	ListClips(ctx context.Context) ([]domain.ClipSegment, error)
}

// Creation Pipeline Ports (Faceless)
type ScriptWriter interface {
	WriteScript(ctx context.Context, niche string, topic string) (*domain.Story, error)
}

type VoiceGenerator interface {
	GenerateVO(ctx context.Context, text string) (path string, err error)
}

type ImageGenerator interface {
	GenerateImage(ctx context.Context, prompt string, sceneID int) (path string, err error)
}

type VideoAssembler interface {
	Assemble(ctx context.Context, story *domain.Story) (path string, err error)
}

type Uploader interface {
	Upload(ctx context.Context, filePath string, title string, description string) error
}

// Clipping Pipeline Ports (Podcast)
type Downloader interface {
	Download(ctx context.Context, url string) (videoPath string, audioPath string, err error)
}

type Transcriber interface {
	Transcribe(ctx context.Context, audioPath string) (text string, words []domain.Word, err error)
}

type StrategistAgent interface {
	Analyze(ctx context.Context, transcript string) ([]domain.ClipSegment, error)
}

type VideoEditor interface {
	CropAndRender(ctx context.Context, videoPath string, segment domain.ClipSegment, faceX int) (outputPath string, err error)
}

type VisionAgent interface {
	DetectFaceCenter(ctx context.Context, videoPath string, startTime string, endTime string) (xCoord int, err error)
}

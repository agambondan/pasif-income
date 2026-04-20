package ports

import (
	"context"
	"github.com/agambondan/pasif-income/internal/core/domain"
)

// Shared Ports
type Storage interface {
	Upload(ctx context.Context, filePath string, objectName string) (url string, err error)
	ListFiles(ctx context.Context, prefix string) ([]string, error)
}

type Repository interface {
	// Clips & Jobs
	SaveClip(ctx context.Context, clip *domain.ClipSegment, sourceID string, s3Path string) error
	UpdateStatus(ctx context.Context, clipID string, status string) error
	ListClips(ctx context.Context) ([]domain.Clip, error)
	CreateJob(ctx context.Context, job *domain.GenerationJob) error
	UpdateJobArtifact(ctx context.Context, jobID string, title string, description string, pinComment string, videoPath string) error
	UpdateJobStatus(ctx context.Context, jobID string, status string, errMsg string) error
	GetJob(ctx context.Context, jobID string) (*domain.GenerationJob, error)
	ListJobs(ctx context.Context) ([]domain.GenerationJob, error)
	CreateDistributionJob(ctx context.Context, job *domain.DistributionJob) error
	ListPendingDistributionJobs(ctx context.Context) ([]domain.DistributionJob, error)
	ListDistributionJobs(ctx context.Context, generationJobID string) ([]domain.DistributionJob, error)
	ListAllDistributionJobs(ctx context.Context, userID int) ([]domain.DistributionJob, error)
	UpdateDistributionJobStatus(ctx context.Context, jobID int, status string, statusDetail string, externalID string, errMsg string) error

	// Users & Auth
	ListUsers(ctx context.Context) ([]domain.User, error)
	CreateUser(ctx context.Context, username, passwordHash string) error
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	CreateSession(ctx context.Context, userID int) (string, error)
	GetUserBySessionToken(ctx context.Context, sessionToken string) (*domain.User, error)
	DeleteSession(ctx context.Context, sessionToken string) error

	// Platforms & Accounts
	ListAllConnectedAccounts(ctx context.Context) ([]domain.ConnectedAccount, error)
	ListConnectedAccounts(ctx context.Context, userID int) ([]domain.ConnectedAccount, error)
	GetConnectedAccountByID(ctx context.Context, accountID string) (*domain.ConnectedAccount, error)
	SaveConnectedAccount(ctx context.Context, acc *domain.ConnectedAccount) error
	DeleteConnectedAccount(ctx context.Context, accountID string) error

	// Analytics
	SaveVideoMetricSnapshot(ctx context.Context, snapshot *domain.VideoMetricSnapshot) error
	ListVideoMetricSnapshots(ctx context.Context, userID int) ([]domain.VideoMetricSnapshot, error)
	ListVideoMetricSnapshotsByJob(ctx context.Context, generationJobID string) ([]domain.VideoMetricSnapshot, error)

	// Community
	SaveCommunityReplyDraft(ctx context.Context, draft *domain.CommunityReplyDraft) error
	ListCommunityReplyDrafts(ctx context.Context, userID int) ([]domain.CommunityReplyDraft, error)
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

type Publisher interface {
	Publish(ctx context.Context, filePath string, title string, description string, account domain.ConnectedAccount, progress func(string)) (externalID string, err error)
}

type CommentResponder interface {
	DraftReply(ctx context.Context, niche, topic, videoTitle, commentText, persona string) (string, error)
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

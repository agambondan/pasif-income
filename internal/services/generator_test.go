package services

import (
	"context"
	"errors"
	"github.com/agambondan/pasif-income/internal/core/domain"
	"sort"
	"testing"
)

// Fakes
type fakeScriptWriter struct {
	err error
}

func (m *fakeScriptWriter) WriteScript(ctx context.Context, niche string, topic string) (*domain.Story, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &domain.Story{
		Title:  "Test Story",
		Script: "Test Script",
		Scenes: []domain.Scene{{Visual: "Test Visual"}},
	}, nil
}

type fakeVoiceGenerator struct {
	err error
}

func (m *fakeVoiceGenerator) GenerateVO(ctx context.Context, text string, voiceType string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "voice.mp3", nil
}

type fakeImageGenerator struct {
	err error
}

func (m *fakeImageGenerator) GenerateImage(ctx context.Context, prompt string, sceneID int) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "image.png", nil
}

type fakeVideoAssembler struct {
	err error
}

func (m *fakeVideoAssembler) Assemble(ctx context.Context, story *domain.Story) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "video.mp4", nil
}

type fakeUploader struct {
	err error
}

func (m *fakeUploader) Upload(ctx context.Context, filePath string, title string, description string) error {
	return m.err
}

type fakeRepository struct {
	connectedAccounts []domain.ConnectedAccount
	agentEvents       []domain.AgentEvent
}

func (r *fakeRepository) SaveClip(ctx context.Context, clip *domain.ClipSegment, sourceID string, s3Path string) error {
	return nil
}
func (r *fakeRepository) UpdateStatus(ctx context.Context, clipID string, status string) error {
	return nil
}
func (r *fakeRepository) ListClips(ctx context.Context) ([]domain.Clip, error) {
	return nil, nil
}
func (r *fakeRepository) CreateJob(ctx context.Context, job *domain.GenerationJob) error {
	return nil
}
func (r *fakeRepository) UpdateJobArtifact(ctx context.Context, jobID string, title string, description string, pinComment string, videoPath string) error {
	return nil
}
func (r *fakeRepository) UpdateJobProgress(ctx context.Context, jobID string, stage string, progress int) error {
	return nil
}
func (r *fakeRepository) UpdateJobStatus(ctx context.Context, jobID string, status string, errMsg string) error {
	return nil
}
func (r *fakeRepository) GetJob(ctx context.Context, jobID string) (*domain.GenerationJob, error) {
	return nil, nil
}
func (r *fakeRepository) ListJobs(ctx context.Context) ([]domain.GenerationJob, error) {
	return nil, nil
}
func (r *fakeRepository) CreateDistributionJob(ctx context.Context, job *domain.DistributionJob) error {
	return nil
}
func (r *fakeRepository) ListPendingDistributionJobs(ctx context.Context) ([]domain.DistributionJob, error) {
	return nil, nil
}
func (r *fakeRepository) ListDistributionJobs(ctx context.Context, generationJobID string) ([]domain.DistributionJob, error) {
	return nil, nil
}
func (r *fakeRepository) ListAllDistributionJobs(ctx context.Context, userID int) ([]domain.DistributionJob, error) {
	return nil, nil
}
func (r *fakeRepository) UpdateDistributionJobStatus(ctx context.Context, jobID int, status string, statusDetail string, externalID string, errMsg string) error {
	return nil
}
func (r *fakeRepository) CancelJob(ctx context.Context, jobID string) error {
	return nil
}
func (r *fakeRepository) ListUsers(ctx context.Context) ([]domain.User, error) {
	return nil, nil
}
func (r *fakeRepository) CreateUser(ctx context.Context, username, passwordHash string) error {
	return nil
}
func (r *fakeRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	return nil, nil
}
func (r *fakeRepository) CreateSession(ctx context.Context, userID int) (string, error) {
	return "", nil
}
func (r *fakeRepository) GetUserBySessionToken(ctx context.Context, sessionToken string) (*domain.User, error) {
	return nil, nil
}
func (r *fakeRepository) DeleteSession(ctx context.Context, sessionToken string) error {
	return nil
}
func (r *fakeRepository) ListAllConnectedAccounts(ctx context.Context) ([]domain.ConnectedAccount, error) {
	return append([]domain.ConnectedAccount(nil), r.connectedAccounts...), nil
}
func (r *fakeRepository) ListConnectedAccounts(ctx context.Context, userID int) ([]domain.ConnectedAccount, error) {
	accs := make([]domain.ConnectedAccount, 0, len(r.connectedAccounts))
	for _, acc := range r.connectedAccounts {
		if acc.UserID == userID {
			accs = append(accs, acc)
		}
	}
	sort.SliceStable(accs, func(i, j int) bool {
		if accs[i].CreatedAt.Equal(accs[j].CreatedAt) {
			return accs[i].ID > accs[j].ID
		}
		return accs[i].CreatedAt.After(accs[j].CreatedAt)
	})
	return accs, nil
}
func (r *fakeRepository) GetConnectedAccountByID(ctx context.Context, accountID string) (*domain.ConnectedAccount, error) {
	for _, acc := range r.connectedAccounts {
		if acc.ID == accountID {
			copy := acc
			return &copy, nil
		}
	}
	return nil, nil
}
func (r *fakeRepository) SaveConnectedAccount(ctx context.Context, acc *domain.ConnectedAccount) error {
	for i, existing := range r.connectedAccounts {
		if existing.ID == acc.ID {
			r.connectedAccounts[i] = *acc
			return nil
		}
		if existing.UserID == acc.UserID &&
			existing.PlatformID == acc.PlatformID &&
			existing.AuthMethod == acc.AuthMethod &&
			existing.Email == acc.Email {
			r.connectedAccounts[i] = *acc
			return nil
		}
	}
	r.connectedAccounts = append(r.connectedAccounts, *acc)
	return nil
}
func (r *fakeRepository) DeleteConnectedAccount(ctx context.Context, accountID string) error {
	return nil
}
func (r *fakeRepository) SaveVideoMetricSnapshot(ctx context.Context, snapshot *domain.VideoMetricSnapshot) error {
	return nil
}
func (r *fakeRepository) ListVideoMetricSnapshots(ctx context.Context, userID int) ([]domain.VideoMetricSnapshot, error) {
	return nil, nil
}
func (r *fakeRepository) ListVideoMetricSnapshotsByJob(ctx context.Context, generationJobID string) ([]domain.VideoMetricSnapshot, error) {
	return nil, nil
}
func (r *fakeRepository) SaveAgentEvent(ctx context.Context, event *domain.AgentEvent) error {
	if event != nil {
		copy := *event
		r.agentEvents = append(r.agentEvents, copy)
	}
	return nil
}
func (r *fakeRepository) ListAgentEvents(ctx context.Context, jobID string) ([]domain.AgentEvent, error) {
	events := make([]domain.AgentEvent, 0, len(r.agentEvents))
	for _, event := range r.agentEvents {
		if event.JobID == jobID {
			events = append(events, event)
		}
	}
	return events, nil
}
func (r *fakeRepository) SaveCommunityReplyDraft(ctx context.Context, draft *domain.CommunityReplyDraft) error {
	return nil
}
func (r *fakeRepository) ListCommunityReplyDrafts(ctx context.Context, userID int) ([]domain.CommunityReplyDraft, error) {
	return nil, nil
}

func TestGenerateContent_Success(t *testing.T) {
	s := NewGeneratorService(
		&fakeRepository{},
		&fakeScriptWriter{},
		nil,
		&fakeVoiceGenerator{},
		&fakeImageGenerator{},
		&fakeVideoAssembler{},
		&fakeUploader{},
		nil,
		nil,
		nil,
		nil,
	)

	story, err := s.GenerateContent(context.Background(), "job-1", "motivation", "discipline", "en-US-Standard-A")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if story == nil || story.VideoOutput != "video.mp4" {
		t.Fatalf("expected story with video path, got %#v", story)
	}
}

func TestGenerateContent_ScriptError(t *testing.T) {
	expectedErr := errors.New("failed to write script")
	s := NewGeneratorService(
		&fakeRepository{},
		&fakeScriptWriter{err: expectedErr},
		nil,
		&fakeVoiceGenerator{},
		&fakeImageGenerator{},
		&fakeVideoAssembler{},
		&fakeUploader{},
		nil,
		nil,
		nil,
		nil,
	)

	_, err := s.GenerateContent(context.Background(), "job-1", "motivation", "discipline", "en-US-Standard-A")
	if err == nil || err.Error() != "script writer (Gemini & Codex): failed to write script" {
		t.Errorf("expected script writer error, got %v", err)
	}
}

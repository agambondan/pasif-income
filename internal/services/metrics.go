package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	yt "google.golang.org/api/youtube/v3"
)

type MetricsService struct {
	repo     ports.Repository
	interval time.Duration
}

func NewMetricsService(repo ports.Repository) *MetricsService {
	return &MetricsService{
		repo:     repo,
		interval: metricsSyncIntervalFromEnv(),
	}
}

func (s *MetricsService) StartWorker(ctx context.Context) {
	if s == nil || s.repo == nil {
		return
	}

	log.Println("--- Metrics Sync Worker Started ---")
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.SyncAll(ctx); err != nil {
				log.Printf("metrics worker: %v\n", err)
			}
		}
	}
}

func (s *MetricsService) SyncAll(ctx context.Context) (int, error) {
	if s == nil || s.repo == nil {
		return 0, fmt.Errorf("repository unavailable")
	}

	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return 0, err
	}

	total := 0
	for _, user := range users {
		synced, err := s.SyncUser(ctx, user.ID)
		if err != nil {
			log.Printf("metrics sync user %d failed: %v\n", user.ID, err)
		}
		total += synced
	}
	return total, nil
}

func (s *MetricsService) SyncUser(ctx context.Context, userID int) (int, error) {
	if s == nil || s.repo == nil {
		return 0, fmt.Errorf("repository unavailable")
	}

	accounts, err := s.repo.ListConnectedAccounts(ctx, userID)
	if err != nil {
		return 0, err
	}
	if len(accounts) == 0 {
		return 0, nil
	}

	jobs, err := s.repo.ListAllDistributionJobs(ctx, userID)
	if err != nil {
		return 0, err
	}

	jobsByAccount := make(map[string][]domain.DistributionJob)
	for _, job := range jobs {
		if !shouldSyncMetricsForJob(job) {
			continue
		}
		jobsByAccount[job.AccountID] = append(jobsByAccount[job.AccountID], job)
	}

	synced := 0
	for _, account := range accounts {
		if account.PlatformID != "youtube" || account.AuthMethod != domain.AuthMethodAPI {
			continue
		}
		accountJobs := jobsByAccount[account.ID]
		if len(accountJobs) == 0 {
			continue
		}

		svc, err := youtubeMetricsServiceForAccount(ctx, account)
		if err != nil {
			log.Printf("metrics youtube service for account %s failed: %v\n", account.ID, err)
			continue
		}

		for _, job := range accountJobs {
			snapshot, err := fetchYouTubeMetricSnapshot(ctx, s.repo, svc, userID, account, job)
			if err != nil {
				log.Printf("metrics fetch failed for job %s: %v\n", job.GenerationJobID, err)
				continue
			}
			if err := s.repo.SaveVideoMetricSnapshot(ctx, snapshot); err != nil {
				return synced, err
			}
			synced++
		}
	}

	return synced, nil
}

func shouldSyncMetricsForJob(job domain.DistributionJob) bool {
	return strings.EqualFold(job.Platform, "youtube") &&
		strings.EqualFold(job.Status, "completed") &&
		strings.TrimSpace(job.ExternalID) != ""
}

func youtubeMetricsServiceForAccount(ctx context.Context, account domain.ConnectedAccount) (*yt.Service, error) {
	cfg, err := youtubeOAuthConfig()
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken:  strings.TrimSpace(account.AccessToken),
		RefreshToken: strings.TrimSpace(account.RefreshToken),
		Expiry:       account.Expiry,
	}
	if token.AccessToken == "" && token.RefreshToken == "" {
		return nil, fmt.Errorf("missing oauth token for account %s", account.ID)
	}

	tokenSource := cfg.TokenSource(ctx, token)
	return yt.NewService(ctx, option.WithTokenSource(tokenSource))
}

func fetchYouTubeMetricSnapshot(ctx context.Context, repo ports.Repository, svc *yt.Service, userID int, account domain.ConnectedAccount, job domain.DistributionJob) (*domain.VideoMetricSnapshot, error) {
	generationJob, err := repo.GetJob(ctx, job.GenerationJobID)
	if err != nil {
		return nil, err
	}

	call := svc.Videos.List([]string{"snippet", "statistics"}).Id(job.ExternalID)
	res, err := call.Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	if len(res.Items) == 0 {
		return nil, fmt.Errorf("video %s not found", job.ExternalID)
	}

	video := res.Items[0]
	stats := video.Statistics
	if stats == nil {
		return nil, fmt.Errorf("video %s has no statistics", job.ExternalID)
	}

	title := ""
	if video.Snippet != nil {
		title = strings.TrimSpace(video.Snippet.Title)
	}
	if title == "" {
		title = strings.TrimSpace(generationJob.Title)
	}

	return &domain.VideoMetricSnapshot{
		UserID:            userID,
		GenerationJobID:   job.GenerationJobID,
		DistributionJobID: job.ID,
		AccountID:         account.ID,
		Platform:          job.Platform,
		Niche:             strings.TrimSpace(generationJob.Niche),
		ExternalID:        job.ExternalID,
		VideoTitle:        title,
		ViewCount:         stats.ViewCount,
		LikeCount:         stats.LikeCount,
		CommentCount:      stats.CommentCount,
		CollectedAt:       time.Now().UTC(),
	}, nil
}

func metricsSyncIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("METRICS_SYNC_INTERVAL_SECONDS"))
	if raw == "" {
		return 24 * time.Hour
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 60 {
		return 24 * time.Hour
	}
	return time.Duration(seconds) * time.Second
}

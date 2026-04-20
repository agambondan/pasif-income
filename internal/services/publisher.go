package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
)

type PublisherService struct {
	repo      ports.Repository
	publisher ports.Publisher
	interval  time.Duration
}

func NewPublisherService(repo ports.Repository, publisher ports.Publisher) *PublisherService {
	return &PublisherService{
		repo:      repo,
		publisher: publisher,
		interval:  5 * time.Second,
	}
}

func (s *PublisherService) StartWorker(ctx context.Context) {
	log.Println("--- Distribution Worker Started ---")
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.processPendingJobs(ctx); err != nil {
				log.Printf("distribution worker: %v\n", err)
			}
		}
	}
}

func (s *PublisherService) processPendingJobs(ctx context.Context) error {
	if s.repo == nil {
		return errors.New("repository unavailable")
	}
	if s.publisher == nil {
		return errors.New("publisher unavailable")
	}

	jobs, err := s.repo.ListPendingDistributionJobs(ctx)
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		log.Println("Distribution Worker: no pending distribution jobs")
		return nil
	}

	for _, job := range jobs {
		if err := s.processJob(ctx, job); err != nil {
			log.Printf("distribution worker: job %d failed: %v\n", job.ID, err)
		}
	}

	return nil
}

func (s *PublisherService) processJob(ctx context.Context, job domain.DistributionJob) error {
	updateDetail := func(detail string) {
		_ = s.repo.UpdateDistributionJobStatus(ctx, job.ID, "uploading", detail, "", "")
	}

	generationJob, err := s.repo.GetJob(ctx, job.GenerationJobID)
	if err != nil {
		_ = s.repo.UpdateDistributionJobStatus(ctx, job.ID, "failed", "loading_generation_job", "", fmt.Sprintf("load generation job: %v", err))
		return err
	}
	updateDetail("loading_generation_job")
	if generationJob.VideoPath == "" {
		err = errors.New("missing video path")
		_ = s.repo.UpdateDistributionJobStatus(ctx, job.ID, "failed", "missing_video_path", "", err.Error())
		return err
	}

	updateDetail("loading_connected_account")
	account, err := s.repo.GetConnectedAccountByID(ctx, job.AccountID)
	if err != nil {
		_ = s.repo.UpdateDistributionJobStatus(ctx, job.ID, "failed", "loading_connected_account", "", fmt.Sprintf("load account: %v", err))
		return err
	}
	if account.PlatformID != job.Platform {
		err = fmt.Errorf("platform mismatch: account=%s job=%s", account.PlatformID, job.Platform)
		_ = s.repo.UpdateDistributionJobStatus(ctx, job.ID, "failed", "platform_mismatch", "", err.Error())
		return err
	}
	if account.UserID == 0 {
		err = errors.New("account owner missing")
		_ = s.repo.UpdateDistributionJobStatus(ctx, job.ID, "failed", "account_owner_missing", "", err.Error())
		return err
	}

	if err := s.repo.UpdateDistributionJobStatus(ctx, job.ID, "uploading", "preparing_publish", "", ""); err != nil {
		return err
	}

	externalID, err := s.publisher.Publish(ctx, generationJob.VideoPath, generationJob.Title, generationJob.Description, *account, func(detail string) {
		_ = s.repo.UpdateDistributionJobStatus(ctx, job.ID, "uploading", detail, "", "")
	})
	if err != nil {
		_ = s.repo.UpdateDistributionJobStatus(ctx, job.ID, "failed", "publish_failed", "", err.Error())
		if failoverJob, failoverErr := s.enqueueFailoverJob(ctx, job, generationJob, account, err); failoverErr != nil {
			log.Printf("distribution failover enqueue failed for job %d: %v\n", job.ID, failoverErr)
		} else if failoverJob != nil {
			log.Printf("distribution failover queued: source=%d target=%d platform=%s account=%s\n", job.ID, failoverJob.ID, failoverJob.Platform, failoverJob.AccountID)
		}
		return err
	}

	if err := s.repo.UpdateDistributionJobStatus(ctx, job.ID, "completed", "publish_complete", externalID, ""); err != nil {
		return err
	}

	log.Printf("Successfully published job %d to %s (%s)\n", job.ID, job.Platform, externalID)
	return nil
}

func (s *PublisherService) enqueueFailoverJob(ctx context.Context, failedJob domain.DistributionJob, generationJob *domain.GenerationJob, account *domain.ConnectedAccount, cause error) (*domain.DistributionJob, error) {
	if !destinationFailoverEnabled() {
		return nil, nil
	}
	if failedJob.RetryAttempt >= destinationFailoverMaxDepth() {
		return nil, nil
	}
	if s.repo == nil {
		return nil, errors.New("repository unavailable")
	}
	if account == nil || generationJob == nil {
		return nil, nil
	}

	accounts, err := s.repo.ListConnectedAccounts(ctx, account.UserID)
	if err != nil {
		return nil, err
	}

	candidates := failoverAccountsForDestination(accounts, *account)
	if len(candidates) == 0 {
		return nil, nil
	}

	next := candidates[0]
	nextJob := &domain.DistributionJob{
		GenerationJobID:  failedJob.GenerationJobID,
		AccountID:        next.ID,
		Platform:         failedJob.Platform,
		Status:           "pending",
		StatusDetail:     fmt.Sprintf("failover_from_%d", failedJob.ID),
		ScheduledAt:      nil,
		RetrySourceJobID: &failedJob.ID,
		RetryAttempt:     failedJob.RetryAttempt + 1,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := s.repo.CreateDistributionJob(ctx, nextJob); err != nil {
		return nil, err
	}
	return nextJob, nil
}

func failoverAccountsForDestination(accounts []domain.ConnectedAccount, current domain.ConnectedAccount) []domain.ConnectedAccount {
	candidates := make([]domain.ConnectedAccount, 0)
	for _, acc := range accounts {
		if acc.ID == current.ID {
			continue
		}
		if !strings.EqualFold(acc.PlatformID, current.PlatformID) {
			continue
		}
		candidates = append(candidates, acc)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		pi := failoverPriority(candidates[i], current)
		pj := failoverPriority(candidates[j], current)
		if pi == pj {
			return candidates[i].CreatedAt.Before(candidates[j].CreatedAt)
		}
		return pi < pj
	})
	return candidates
}

func failoverPriority(candidate, current domain.ConnectedAccount) int {
	if strings.EqualFold(candidate.AuthMethod, current.AuthMethod) {
		return 0
	}
	if strings.EqualFold(current.AuthMethod, domain.AuthMethodAPI) && strings.EqualFold(candidate.AuthMethod, domain.AuthMethodChromiumProfile) {
		return 1
	}
	if strings.EqualFold(candidate.AuthMethod, domain.AuthMethodAPI) {
		return 2
	}
	return 3
}

func destinationFailoverEnabled() bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("DESTINATION_FAILOVER_ENABLED")))
	return raw == "" || raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

func destinationFailoverMaxDepth() int {
	raw := strings.TrimSpace(os.Getenv("DESTINATION_FAILOVER_MAX_DEPTH"))
	if raw == "" {
		return 1
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 1
	}
	return value
}

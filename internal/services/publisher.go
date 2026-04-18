package services

import (
	"context"
	"errors"
	"fmt"
	"log"
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
		return err
	}

	if err := s.repo.UpdateDistributionJobStatus(ctx, job.ID, "completed", "publish_complete", externalID, ""); err != nil {
		return err
	}

	log.Printf("Successfully published job %d to %s (%s)\n", job.ID, job.Platform, externalID)
	return nil
}

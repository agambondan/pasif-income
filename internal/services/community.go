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
	yt "google.golang.org/api/youtube/v3"
)

type CommunityService struct {
	repo                ports.Repository
	responder           ports.CommentResponder
	interval            time.Duration
	maxCommentsPerVideo int
	autoReply           bool
}

func NewCommunityService(repo ports.Repository, responder ports.CommentResponder) *CommunityService {
	return &CommunityService{
		repo:                repo,
		responder:           responder,
		interval:            communitySyncIntervalFromEnv(),
		maxCommentsPerVideo: communityMaxCommentsPerVideoFromEnv(),
		autoReply:           communityAutoReplyEnabledFromEnv(),
	}
}

func (s *CommunityService) StartWorker(ctx context.Context) {
	if s == nil || s.repo == nil || s.responder == nil {
		return
	}

	log.Println("--- Community Sync Worker Started ---")
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.SyncAll(ctx); err != nil {
				log.Printf("community worker: %v\n", err)
			}
		}
	}
}

func (s *CommunityService) SyncAll(ctx context.Context) (int, error) {
	if s == nil || s.repo == nil || s.responder == nil {
		return 0, fmt.Errorf("community service unavailable")
	}

	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return 0, err
	}

	total := 0
	for _, user := range users {
		synced, err := s.SyncUser(ctx, user.ID)
		if err != nil {
			log.Printf("community sync user %d failed: %v\n", user.ID, err)
		}
		total += synced
	}
	return total, nil
}

func (s *CommunityService) SyncUser(ctx context.Context, userID int) (int, error) {
	if s == nil || s.repo == nil || s.responder == nil {
		return 0, fmt.Errorf("community service unavailable")
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
		if !shouldSyncCommunityForJob(job) {
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
			log.Printf("community youtube service for account %s failed: %v\n", account.ID, err)
			continue
		}

		for _, job := range accountJobs {
			count, err := s.syncYouTubeCommentsForJob(ctx, userID, account, job, svc)
			if err != nil {
				log.Printf("community sync failed for job %s: %v\n", job.GenerationJobID, err)
				continue
			}
			synced += count
		}
	}

	return synced, nil
}

func shouldSyncCommunityForJob(job domain.DistributionJob) bool {
	return strings.EqualFold(job.Platform, "youtube") &&
		strings.EqualFold(job.Status, "completed") &&
		strings.TrimSpace(job.ExternalID) != ""
}

func (s *CommunityService) syncYouTubeCommentsForJob(ctx context.Context, userID int, account domain.ConnectedAccount, job domain.DistributionJob, svc *yt.Service) (int, error) {
	generationJob, err := s.repo.GetJob(ctx, job.GenerationJobID)
	if err != nil {
		return 0, err
	}

	maxResults := int64(s.maxCommentsPerVideo)
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 100 {
		maxResults = 100
	}

	call := svc.CommentThreads.List([]string{"snippet"}).
		VideoId(job.ExternalID).
		Order("time").
		TextFormat("plainText").
		MaxResults(maxResults)

	res, err := call.Context(ctx).Do()
	if err != nil {
		return 0, err
	}

	persona := strings.TrimSpace(os.Getenv("BRAND_PERSONA"))
	if persona == "" {
		persona = brandPersonaForNiche(generationJob.Niche)
	}

	synced := 0
	for _, thread := range res.Items {
		if thread == nil || thread.Snippet == nil || thread.Snippet.TopLevelComment == nil {
			continue
		}
		if thread.Snippet.TotalReplyCount > 0 {
			continue
		}

		comment := thread.Snippet.TopLevelComment
		commentSnippet := comment.Snippet
		if commentSnippet == nil {
			continue
		}

		commentText := strings.TrimSpace(commentSnippet.TextDisplay)
		if commentText == "" {
			commentText = strings.TrimSpace(commentSnippet.TextOriginal)
		}
		if commentText == "" {
			continue
		}

		reply, err := s.responder.DraftReply(ctx, generationJob.Niche, generationJob.Topic, generationJob.Title, commentText, persona)
		if err != nil {
			log.Printf("community draft reply failed for comment %s: %v\n", comment.Id, err)
			continue
		}
		reply = strings.TrimSpace(reply)
		if reply == "" {
			continue
		}

		status := "draft"
		postedID := ""
		var repliedAt *time.Time

		if s.autoReply {
			postedID, err = postYouTubeReply(ctx, svc, comment.Id, reply)
			if err != nil {
				log.Printf("community auto reply failed for comment %s: %v\n", comment.Id, err)
			} else {
				status = "replied"
				now := time.Now().UTC()
				repliedAt = &now
			}
		}

		draft := &domain.CommunityReplyDraft{
			UserID:            userID,
			GenerationJobID:   job.GenerationJobID,
			DistributionJobID: job.ID,
			AccountID:         account.ID,
			Platform:          job.Platform,
			Niche:             strings.TrimSpace(generationJob.Niche),
			VideoTitle:        firstNonEmpty(strings.TrimSpace(generationJob.Title), strings.TrimSpace(job.ExternalID)),
			ExternalCommentID: comment.Id,
			ParentCommentID:   commentSnippet.ParentId,
			CommentAuthor:     strings.TrimSpace(commentSnippet.AuthorDisplayName),
			CommentText:       commentText,
			SuggestedReply:    reply,
			Status:            status,
			PostedExternalID:  postedID,
			RepliedAt:         repliedAt,
		}
		if err := s.repo.SaveCommunityReplyDraft(ctx, draft); err != nil {
			return synced, err
		}
		synced++
	}

	return synced, nil
}

func postYouTubeReply(ctx context.Context, svc *yt.Service, parentCommentID, reply string) (string, error) {
	resp, err := svc.Comments.Insert([]string{"snippet"}, &yt.Comment{
		Snippet: &yt.CommentSnippet{
			ParentId:     parentCommentID,
			TextOriginal: reply,
		},
	}).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	if resp == nil || strings.TrimSpace(resp.Id) == "" {
		return "", fmt.Errorf("youtube reply returned empty id")
	}
	return resp.Id, nil
}

func communitySyncIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("COMMUNITY_SYNC_INTERVAL_SECONDS"))
	if raw == "" {
		return 24 * time.Hour
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 60 {
		return 24 * time.Hour
	}
	return time.Duration(seconds) * time.Second
}

func communityMaxCommentsPerVideoFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("COMMUNITY_MAX_COMMENTS_PER_VIDEO"))
	if raw == "" {
		return 10
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 10
	}
	return n
}

func communityAutoReplyEnabledFromEnv() bool {
	raw := strings.TrimSpace(os.Getenv("COMMUNITY_AUTO_REPLY_ENABLED"))
	if raw == "" {
		return false
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

package adapters

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

type uploadAdapter interface {
	Publish(ctx context.Context, filePath, title, description string, account domain.ConnectedAccount, progress func(string)) (string, error)
}

type DistributionPublisher struct {
	youtubeAPI  uploadAdapter
	browser     uploadAdapter
	retryLimit  map[string]int
	fallbackAny bool
}

func NewDistributionPublisher() *DistributionPublisher {
	return &DistributionPublisher{
		youtubeAPI:  NewYouTubeAPIUploadAdapter(),
		browser:     NewBrowserProfileUploadAdapter(NewChromiumRunnerFromEnv()),
		retryLimit:  browserRetryLimitsFromEnv(),
		fallbackAny: fallbackEnabled(),
	}
}

func (p *DistributionPublisher) Publish(ctx context.Context, filePath, title, description string, account domain.ConnectedAccount, progress func(string)) (string, error) {
	primary, fallback := p.publishersFor(account)

	externalID, err := p.publishWithRetry(ctx, primary, filePath, title, description, account, progress)
	if err == nil {
		return externalID, nil
	}
	if !p.fallbackAny || fallback == nil {
		return "", err
	}

	log.Printf("Primary publish failed for %s, trying fallback: %v\n", account.ID, err)
	if progress != nil {
		progress("retrying_with_fallback")
	}
	return p.publishWithRetry(ctx, fallback, filePath, title, description, account, progress)
}

func (p *DistributionPublisher) publishersFor(account domain.ConnectedAccount) (uploadAdapter, uploadAdapter) {
	if account.PlatformID == "youtube" && account.AuthMethod == domain.AuthMethodAPI {
		return p.youtubeAPI, p.browser
	}
	if account.PlatformID == "youtube" {
		return p.browser, p.youtubeAPI
	}
	return p.browser, nil
}

func buildExternalID(prefix, title, email string) string {
	value := strings.ToLower(strings.TrimSpace(title + "_" + email))
	replacer := strings.NewReplacer("@", "_at_", ".", "_", " ", "_", "/", "_", "\\", "_", ":", "_")
	slug := replacer.Replace(value)
	if slug == "" {
		slug = "post"
	}
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().Unix(), slug)
}

func (p *DistributionPublisher) publishWithRetry(ctx context.Context, fn uploadAdapter, filePath, title, description string, account domain.ConnectedAccount, progress func(string)) (string, error) {
	if fn == nil {
		return "", fmt.Errorf("publisher unavailable")
	}

	limit := p.retryLimitFor(account.PlatformID)
	if limit < 1 {
		limit = 1
	}

	var lastErr error
	for attempt := 1; attempt <= limit; attempt++ {
		if progress != nil {
			progress(fmt.Sprintf("attempt_%d_of_%d", attempt, limit))
		}
		externalID, err := fn.Publish(ctx, filePath, title, description, account, progress)
		if err == nil {
			return externalID, nil
		}
		lastErr = err
		log.Printf("publish attempt %d/%d failed for %s: %v\n", attempt, limit, account.ID, err)
		if attempt < limit {
			delay := time.Duration(attempt) * 2 * time.Second
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return "", ctx.Err()
			case <-timer.C:
			}
		}
	}

	return "", lastErr
}

func (p *DistributionPublisher) retryLimitFor(platformID string) int {
	if p == nil || p.retryLimit == nil {
		return defaultRetryLimit()
	}
	if value, ok := p.retryLimit[strings.ToLower(strings.TrimSpace(platformID))]; ok {
		return value
	}
	return defaultRetryLimit()
}

func browserRetryLimitsFromEnv() map[string]int {
	limits := map[string]int{
		"default": defaultRetryLimit(),
	}
	for _, platform := range []string{"youtube", "tiktok", "instagram"} {
		limits[platform] = platformRetryLimit(platform)
	}
	return limits
}

func defaultRetryLimit() int {
	if value := retryLimitFromEnv("PUBLISH_RETRY_LIMIT_DEFAULT", 0); value > 0 {
		return value
	}
	return retryLimitFromEnv("PUBLISH_RETRY_LIMIT", 3)
}

func platformRetryLimit(platform string) int {
	key := fmt.Sprintf("PUBLISH_RETRY_LIMIT_%s", strings.ToUpper(platform))
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		return retryLimitFromEnv(key, 3)
	}
	return defaultRetryLimit()
}

func retryLimitFromEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func fallbackEnabled() bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("PUBLISH_FALLBACK_ENABLED")))
	return raw == "" || raw == "1" || raw == "true" || raw == "yes"
}

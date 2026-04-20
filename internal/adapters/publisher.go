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
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	yt "google.golang.org/api/youtube/v3"
)

type DistributionPublisher struct {
	browser     *ChromiumRunner
	retryLimit  map[string]int
	fallbackAny bool
}

type publishFn func(context.Context, string, string, string, domain.ConnectedAccount, func(string)) (string, error)

func NewDistributionPublisher() *DistributionPublisher {
	return &DistributionPublisher{
		browser:     NewChromiumRunnerFromEnv(),
		retryLimit:  browserRetryLimitsFromEnv(),
		fallbackAny: fallbackEnabled(),
	}
}

func (p *DistributionPublisher) Publish(ctx context.Context, filePath, title, description string, account domain.ConnectedAccount, progress func(string)) (string, error) {
	var primary publishFn
	var fallback publishFn

	switch {
	case account.PlatformID == "youtube" && account.AuthMethod == domain.AuthMethodAPI:
		primary = p.publishYouTubeAPI
		fallback = p.publishChromiumProfile
	default:
		primary = p.publishChromiumProfile
		if account.PlatformID == "youtube" {
			fallback = p.publishYouTubeAPI
		}
	}

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

func (p *DistributionPublisher) publishYouTubeAPI(ctx context.Context, filePath, title, description string, account domain.ConnectedAccount, progress func(string)) (string, error) {
	accessToken := strings.TrimSpace(account.AccessToken)
	if accessToken == "" {
		accessToken = strings.TrimSpace(os.Getenv("YOUTUBE_ACCESS_TOKEN"))
	}
	if accessToken == "" {
		return "", fmt.Errorf("missing youtube access token for account %s", account.ID)
	}

	// For API uploads, we only proceed when credentials are ready.
	log.Printf("Publishing %s to YouTube API for account %s\n", filePath, account.DisplayName)
	log.Printf("Title: %s\n", title)
	log.Printf("Description: %s\n", description)
	if progress != nil {
		progress("api_initializing")
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	svc, err := yt.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return "", err
	}
	if progress != nil {
		progress("api_uploading")
	}

	fh, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer fh.Close()

	privacyStatus := strings.TrimSpace(os.Getenv("YOUTUBE_PRIVACY_STATUS"))
	if privacyStatus == "" {
		privacyStatus = "private"
	}

	video := &yt.Video{
		Snippet: &yt.VideoSnippet{
			Title:       title,
			Description: description,
			CategoryId:  "22",
		},
		Status: &yt.VideoStatus{
			PrivacyStatus: privacyStatus,
		},
	}

	call := svc.Videos.Insert([]string{"snippet", "status"}, video)
	call.Media(fh)

	uploaded, err := call.Context(ctx).Do()
	if err != nil {
		return "", err
	}
	if uploaded == nil {
		return "", fmt.Errorf("youtube upload returned empty response")
	}
	if progress != nil {
		progress("api_completed")
	}

	return uploaded.Id, nil
}

func (p *DistributionPublisher) publishChromiumProfile(ctx context.Context, filePath, title, description string, account domain.ConnectedAccount, progress func(string)) (string, error) {
	if account.ProfilePath == "" {
		return "", fmt.Errorf("missing chromium profile path for account %s", account.ID)
	}

	log.Printf("Publishing %s via Chromium profile for %s\n", filePath, account.ProfilePath)

	if progress != nil {
		progress("profile_ready")
	}

	url := browserTargetURL(account.PlatformID)
	if url == "" {
		return "", fmt.Errorf("missing browser target url for platform %s", account.PlatformID)
	}

	if p.browser == nil {
		return "", fmt.Errorf("chromium runner unavailable")
	}

	if err := p.browser.AutomateUpload(ctx, account.ProfilePath, url, filePath, title, description, account.PlatformID, progress); err != nil {
		return "", err
	}

	return buildExternalID("browser", title, account.Email), nil
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

func (p *DistributionPublisher) publishWithRetry(ctx context.Context, fn publishFn, filePath, title, description string, account domain.ConnectedAccount, progress func(string)) (string, error) {
	limit := p.retryLimitFor(account.PlatformID)
	if limit < 1 {
		limit = 1
	}

	var lastErr error
	for attempt := 1; attempt <= limit; attempt++ {
		if progress != nil {
			progress(fmt.Sprintf("attempt_%d_of_%d", attempt, limit))
		}
		externalID, err := fn(ctx, filePath, title, description, account, progress)
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

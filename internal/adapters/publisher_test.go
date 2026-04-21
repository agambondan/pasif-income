package adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

type fakeUploadAdapter struct {
	calls int
	err   error
	id    string
}

func (f *fakeUploadAdapter) Publish(ctx context.Context, filePath, title, description string, account domain.ConnectedAccount, progress func(string)) (string, error) {
	_ = ctx
	f.calls++
	if progress != nil {
		progress("fake")
	}
	if f.err != nil {
		return "", f.err
	}
	if f.id == "" {
		return "ok", nil
	}
	return f.id, nil
}

func TestDistributionPublisherRoutesYouTubeAPI(t *testing.T) {
	t.Parallel()

	api := &fakeUploadAdapter{id: "yt-api"}
	browser := &fakeUploadAdapter{id: "browser"}
	pub := &DistributionPublisher{
		youtubeAPI:  api,
		browser:     browser,
		retryLimit:  map[string]int{"youtube": 1, "default": 1},
		fallbackAny: false,
	}

	account := domain.ConnectedAccount{
		ID:         "acc-1",
		PlatformID: "youtube",
		AuthMethod: domain.AuthMethodAPI,
	}

	got, err := pub.Publish(context.Background(), "/tmp/video.mp4", "title", "desc", account, nil)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if got != "yt-api" {
		t.Fatalf("unexpected external id: %s", got)
	}
	if api.calls != 1 {
		t.Fatalf("expected 1 API call, got %d", api.calls)
	}
	if browser.calls != 0 {
		t.Fatalf("expected browser adapter to stay idle, got %d calls", browser.calls)
	}
}

func TestDistributionPublisherRoutesBrowserForTikTok(t *testing.T) {
	t.Parallel()

	api := &fakeUploadAdapter{id: "yt-api"}
	browser := &fakeUploadAdapter{id: "browser"}
	pub := &DistributionPublisher{
		youtubeAPI:  api,
		browser:     browser,
		retryLimit:  map[string]int{"tiktok": 1, "default": 1},
		fallbackAny: false,
	}

	account := domain.ConnectedAccount{
		ID:          "acc-2",
		PlatformID:  "tiktok",
		AuthMethod:  domain.AuthMethodChromiumProfile,
		ProfilePath: "/profiles/tiktok/acc-2",
		Email:       "creator@example.com",
	}

	got, err := pub.Publish(context.Background(), "/tmp/video.mp4", "title", "desc", account, nil)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if got != "browser" {
		t.Fatalf("unexpected external id: %s", got)
	}
	if api.calls != 0 {
		t.Fatalf("expected API adapter to stay idle, got %d calls", api.calls)
	}
	if browser.calls != 1 {
		t.Fatalf("expected browser adapter to be called once, got %d", browser.calls)
	}
}

func TestDistributionPublisherRoutesBrowserForInstagram(t *testing.T) {
	t.Parallel()

	api := &fakeUploadAdapter{id: "yt-api"}
	browser := &fakeUploadAdapter{id: "browser"}
	pub := &DistributionPublisher{
		youtubeAPI:  api,
		browser:     browser,
		retryLimit:  map[string]int{"instagram": 1, "default": 1},
		fallbackAny: false,
	}

	account := domain.ConnectedAccount{
		ID:          "acc-4",
		PlatformID:  "instagram",
		AuthMethod:  domain.AuthMethodChromiumProfile,
		ProfilePath: "/profiles/instagram/acc-4",
		Email:       "creator@example.com",
	}

	got, err := pub.Publish(context.Background(), "/tmp/video.mp4", "title", "desc", account, nil)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if got != "browser" {
		t.Fatalf("unexpected external id: %s", got)
	}
	if api.calls != 0 {
		t.Fatalf("expected API adapter to stay idle, got %d calls", api.calls)
	}
	if browser.calls != 1 {
		t.Fatalf("expected browser adapter to be called once, got %d", browser.calls)
	}
}

func TestDistributionPublisherFallbacksFromYouTubeAPIToBrowser(t *testing.T) {
	t.Parallel()

	api := &fakeUploadAdapter{err: errors.New("api failed")}
	browser := &fakeUploadAdapter{id: "browser"}
	pub := &DistributionPublisher{
		youtubeAPI:  api,
		browser:     browser,
		retryLimit:  map[string]int{"youtube": 1, "default": 1},
		fallbackAny: true,
	}

	account := domain.ConnectedAccount{
		ID:          "acc-3",
		PlatformID:  "youtube",
		AuthMethod:  domain.AuthMethodAPI,
		ProfilePath: "/profiles/youtube/acc-3",
		Email:       "creator@example.com",
	}

	got, err := pub.Publish(context.Background(), "/tmp/video.mp4", "title", "desc", account, nil)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if got != "browser" {
		t.Fatalf("unexpected external id: %s", got)
	}
	if api.calls != 1 {
		t.Fatalf("expected API adapter to be called once, got %d", api.calls)
	}
	if browser.calls != 1 {
		t.Fatalf("expected browser adapter fallback to be called once, got %d", browser.calls)
	}
}

func TestBrowserTargetURLPerPlatform(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"youtube":     "https://www.youtube.com/upload",
		"tiktok":      "https://www.tiktok.com/upload?lang=en",
		"instagram":   "https://www.instagram.com/create/select/",
		"unsupported": "",
	}

	for platformID, want := range tests {
		if got := browserTargetURL(platformID); got != want {
			t.Fatalf("browserTargetURL(%q) = %q, want %q", platformID, got, want)
		}
	}
}

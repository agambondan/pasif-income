package adapters

import (
	"context"
	"fmt"
	"log"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

type browserAutomator interface {
	AutomateUpload(ctx context.Context, profilePath, targetURL, filePath, title, description, platformID string, progress func(string)) error
}

type BrowserProfileUploadAdapter struct {
	browser browserAutomator
}

func NewBrowserProfileUploadAdapter(browser browserAutomator) *BrowserProfileUploadAdapter {
	return &BrowserProfileUploadAdapter{browser: browser}
}

func (a *BrowserProfileUploadAdapter) Publish(ctx context.Context, filePath, title, description string, account domain.ConnectedAccount, progress func(string)) (string, error) {
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

	if a.browser == nil {
		return "", fmt.Errorf("chromium runner unavailable")
	}

	if err := a.browser.AutomateUpload(ctx, account.ProfilePath, url, filePath, title, description, account.PlatformID, progress); err != nil {
		return "", err
	}

	return buildExternalID("browser", title, account.Email), nil
}

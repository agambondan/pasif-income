package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

func TestLinkConnectedAccountIsIdempotentByEmail(t *testing.T) {
	t.Setenv("CHROMIUM_PROFILE_DIR", t.TempDir())

	repo := &fakeRepository{
		connectedAccounts: []domain.ConnectedAccount{
			{
				ID:          "acc-1",
				UserID:      1,
				PlatformID:  "youtube",
				DisplayName: "Old Name",
				AuthMethod:  domain.AuthMethodChromiumProfile,
				Email:       "agambondan@example.com",
				ProfilePath: "old/path",
				CreatedAt:   time.Now().Add(-time.Hour),
			},
		},
	}
	svc := NewAuthService(repo)

	acc, err := svc.LinkConnectedAccount(
		context.Background(),
		1,
		"youtube",
		"New Display",
		"  AGAMBONDAN@Example.com  ",
		domain.AuthMethodChromiumProfile,
		"",
		"",
		time.Time{},
	)
	if err != nil {
		t.Fatalf("LinkConnectedAccount returned error: %v", err)
	}

	if acc.ID != "acc-1" {
		t.Fatalf("expected existing account id to be reused, got %q", acc.ID)
	}
	if got := len(repo.connectedAccounts); got != 1 {
		t.Fatalf("expected one connected account, got %d", got)
	}
	if got := repo.connectedAccounts[0].Email; got != "agambondan@example.com" {
		t.Fatalf("expected normalized email to be stored, got %q", got)
	}
	if got := repo.connectedAccounts[0].DisplayName; got != "New Display" {
		t.Fatalf("expected display name to be updated, got %q", got)
	}
}

func TestChromiumProfilePathPrefersBrowserProfilesDir(t *testing.T) {
	browserRoot := t.TempDir()
	legacyRoot := t.TempDir()
	t.Setenv("BROWSER_PROFILES_DIR", browserRoot)
	t.Setenv("CHROMIUM_PROFILE_DIR", legacyRoot)

	got := ChromiumProfilePath("youtube", "demo@example.com")
	want := filepath.Join(browserRoot, "youtube", "demo_at_example_com")
	if got != want {
		t.Fatalf("expected browser profiles dir to win, got %q want %q", got, want)
	}
}

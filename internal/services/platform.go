package services

import (
	"context"
	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
)

type PlatformService struct {
	repo ports.Repository
}

func NewPlatformService(repo ports.Repository) *PlatformService {
	return &PlatformService{repo}
}

func (s *PlatformService) GetSupportedPlatforms() []domain.Platform {
	return []domain.Platform{
		{ID: "youtube", Name: "YouTube", Methods: []string{domain.AuthMethodAPI, domain.AuthMethodChromiumProfile}, Description: "Upload to YouTube via OAuth API or Chromium profile automation"},
		{ID: "tiktok", Name: "TikTok", Methods: []string{domain.AuthMethodAPI, domain.AuthMethodChromiumProfile}, Description: "Link a persistent Chromium profile or manual API key for TikTok"},
		{ID: "instagram", Name: "Instagram", Methods: []string{domain.AuthMethodAPI, domain.AuthMethodChromiumProfile}, Description: "Link a persistent Chromium profile or manual API key for Instagram"},
	}
}

func (s *PlatformService) ListConnectedAccounts(ctx context.Context, userID int) ([]domain.ConnectedAccount, error) {
	return s.repo.ListConnectedAccounts(ctx, userID)
}

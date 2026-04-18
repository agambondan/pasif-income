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
		{ID: "youtube", Name: "YouTube", Methods: []string{domain.AuthMethodChromiumProfile, domain.AuthMethodAPI}, Description: "Upload to YouTube Shorts via API or Chromium profile automation"},
		{ID: "tiktok", Name: "TikTok", Methods: []string{domain.AuthMethodChromiumProfile, domain.AuthMethodAPI}, Description: "Upload to TikTok via API or Chromium profile automation"},
		{ID: "instagram", Name: "Instagram", Methods: []string{domain.AuthMethodChromiumProfile, domain.AuthMethodAPI}, Description: "Upload to Instagram Reels via API or Chromium profile automation"},
	}
}

func (s *PlatformService) ListConnectedAccounts(ctx context.Context, userID int) ([]domain.ConnectedAccount, error) {
	return s.repo.ListConnectedAccounts(ctx, userID)
}

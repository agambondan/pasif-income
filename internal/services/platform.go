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
		{ID: "youtube", Name: "YouTube", AuthType: "oauth", Description: "Upload to YouTube Shorts"},
		{ID: "tiktok", Name: "TikTok", AuthType: "oauth", Description: "Upload to TikTok"},
		{ID: "instagram", Name: "Instagram", AuthType: "oauth", Description: "Upload to Instagram Reels"},
	}
}

func (s *PlatformService) ListConnectedAccounts(ctx context.Context, userID int) ([]domain.ConnectedAccount, error) {
	return s.repo.ListConnectedAccounts(ctx, userID)
}

package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo ports.Repository
}

func NewAuthService(repo ports.Repository) *AuthService {
	return &AuthService{repo}
}

func (s *AuthService) Register(ctx context.Context, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.CreateUser(ctx, username, string(hash))
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*domain.User, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

func (s *AuthService) LinkConnectedAccount(ctx context.Context, userID int, platformID, displayName, email, authMethod string) (*domain.ConnectedAccount, error) {
	if authMethod == "" {
		authMethod = domain.AuthMethodChromiumProfile
	}

	acc := &domain.ConnectedAccount{
		ID:          fmt.Sprintf("%s-%s-%d", platformID, sanitizeForID(email), time.Now().UnixNano()),
		UserID:      userID,
		PlatformID:  platformID,
		DisplayName: displayName,
		AuthMethod:  authMethod,
		Email:       email,
		CreatedAt:   time.Now(),
	}

	if authMethod == domain.AuthMethodChromiumProfile {
		profilePath, err := s.ProvisionChromiumProfile(ctx, platformID, email)
		if err != nil {
			return nil, err
		}
		acc.ProfilePath = profilePath
	}

	if err := s.repo.SaveConnectedAccount(ctx, acc); err != nil {
		return nil, err
	}

	return acc, nil
}

func ChromiumProfilePath(platformID, email string) string {
	base := sanitizeForID(email)
	if base == "" {
		base = "unknown"
	}
	root := os.Getenv("CHROMIUM_PROFILE_DIR")
	if root == "" {
		root = "chromium-profiles"
	}
	return filepath.Join(root, platformID, base)
}

func (s *AuthService) ProvisionChromiumProfile(ctx context.Context, platformID, email string) (string, error) {
	path := ChromiumProfilePath(platformID, email)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	meta := map[string]any{
		"platform_id":  platformID,
		"email":        email,
		"profile_path": path,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
	}
	metaPath := filepath.Join(path, "profile.json")
	if data, err := json.MarshalIndent(meta, "", "  "); err == nil {
		_ = os.WriteFile(metaPath, data, 0o644)
	}
	return path, nil
}

func (s *AuthService) CreateSession(ctx context.Context, userID int) (string, error) {
	return s.repo.CreateSession(ctx, userID)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sanitizeForID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer(
		"@", "_at_",
		".", "_",
		"+", "_plus_",
		"/", "_",
		"\\", "_",
		":", "_",
	)
	return replacer.Replace(value)
}

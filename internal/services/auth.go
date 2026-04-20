package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
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

func (s *AuthService) LinkConnectedAccount(ctx context.Context, userID int, platformID, displayName, email, authMethod, accessToken, refreshToken string, expiry time.Time) (*domain.ConnectedAccount, error) {
	if authMethod == "" {
		authMethod = domain.AuthMethodChromiumProfile
	}

	email = strings.TrimSpace(email)
	displayName = strings.TrimSpace(displayName)
	platformID = strings.TrimSpace(platformID)
	authMethod = strings.TrimSpace(authMethod)
	accessToken = strings.TrimSpace(accessToken)
	refreshToken = strings.TrimSpace(refreshToken)

	if platformID == "" {
		return nil, errors.New("platform is required")
	}
	if displayName == "" {
		displayName = strings.ToUpper(platformID)
	}

	acc := &domain.ConnectedAccount{
		ID:           fmt.Sprintf("%s-%s-%d", platformID, sanitizeForID(email), time.Now().UnixNano()),
		UserID:       userID,
		PlatformID:   platformID,
		DisplayName:  displayName,
		AuthMethod:   authMethod,
		Email:        email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiry,
		CreatedAt:    time.Now(),
	}

	if authMethod == domain.AuthMethodChromiumProfile {
		if email == "" {
			return nil, errors.New("email is required for chromium profile linking")
		}
		profilePath, err := s.ProvisionChromiumProfile(ctx, platformID, email)
		if err != nil {
			return nil, err
		}
		acc.ProfilePath = profilePath
	}

	if authMethod == domain.AuthMethodAPI {
		if platformID != "youtube" {
			return nil, fmt.Errorf("api auth is only wired for youtube right now")
		}
		if acc.AccessToken == "" {
			return nil, errors.New("access token is required for youtube api linking")
		}
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

func (s *AuthService) RefreshAccountToken(ctx context.Context, accountID string, userID int) (*domain.ConnectedAccount, error) {
	acc, err := s.repo.GetConnectedAccountByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if acc.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	if acc.AuthMethod != domain.AuthMethodAPI {
		return nil, errors.New("only API accounts can be refreshed")
	}

	token, err := s.refreshOAuthToken(ctx, acc)
	if err != nil {
		return nil, err
	}
	acc.AccessToken = token.AccessToken
	acc.RefreshToken = token.RefreshToken
	acc.Expiry = token.Expiry

	if err := s.repo.SaveConnectedAccount(ctx, acc); err != nil {
		return nil, err
	}
	return acc, nil
}

func (s *AuthService) RevokeAccountToken(ctx context.Context, accountID string, userID int) error {
	acc, err := s.repo.GetConnectedAccountByID(ctx, accountID)
	if err != nil {
		return err
	}
	if acc.UserID != userID {
		return errors.New("unauthorized")
	}

	if acc.AuthMethod == domain.AuthMethodAPI {
		if err := s.revokeOAuthToken(ctx, acc); err != nil {
			return err
		}
		acc.AccessToken = ""
		acc.RefreshToken = ""
		acc.Expiry = time.Time{}
		if err := s.repo.SaveConnectedAccount(ctx, acc); err != nil {
			return err
		}
	}
	return nil
}

func (s *AuthService) refreshOAuthToken(ctx context.Context, acc *domain.ConnectedAccount) (*oauth2.Token, error) {
	cfg, err := youtubeOAuthConfig()
	if err != nil {
		return nil, err
	}
	if acc.RefreshToken == "" {
		return nil, errors.New("refresh token is required")
	}

	token, err := cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: acc.RefreshToken}).Token()
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (s *AuthService) revokeOAuthToken(ctx context.Context, acc *domain.ConnectedAccount) error {
	token := strings.TrimSpace(acc.RefreshToken)
	if token == "" {
		token = strings.TrimSpace(acc.AccessToken)
	}
	if token == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/revoke", strings.NewReader("token="+token))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("revoke token returned %s", resp.Status)
	}
	return nil
}

func youtubeOAuthConfig() (*oauth2.Config, error) {
	clientID := strings.TrimSpace(os.Getenv("YOUTUBE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("YOUTUBE_CLIENT_SECRET"))
	redirectURL := strings.TrimSpace(os.Getenv("YOUTUBE_REDIRECT_URL"))
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, errors.New("youtube oauth config is incomplete; set YOUTUBE_CLIENT_ID, YOUTUBE_CLIENT_SECRET, and YOUTUBE_REDIRECT_URL")
	}

	scopes := youtubeOAuthScopes()
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint:     oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token"},
	}, nil
}

func youtubeOAuthScopes() []string {
	raw := strings.TrimSpace(os.Getenv("YOUTUBE_SCOPES"))
	if raw == "" {
		return []string{
			"https://www.googleapis.com/auth/youtube.upload",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		}
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' '
	})
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			scopes = append(scopes, trimmed)
		}
	}
	return scopes
}

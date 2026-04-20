package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
)

type BrandingService struct {
	image    ports.ImageGenerator
	assetDir string
	enabled  bool
}

func NewBrandingService(image ports.ImageGenerator) *BrandingService {
	assetDir := strings.TrimSpace(os.Getenv("BRANDING_ASSET_DIR"))
	if assetDir == "" {
		assetDir = "branding-assets"
	}
	enabled := true
	if raw := strings.TrimSpace(os.Getenv("BRANDING_ENABLED")); raw != "" {
		switch strings.ToLower(raw) {
		case "0", "false", "no", "off":
			enabled = false
		}
	}
	return &BrandingService{image: image, assetDir: assetDir, enabled: enabled}
}

func (s *BrandingService) Resolve(ctx context.Context, niche string) (*domain.BrandProfile, error) {
	if s == nil || !s.enabled {
		return nil, nil
	}

	slug := brandSlug(niche)
	if slug == "" {
		slug = "default"
	}

	persona := strings.TrimSpace(os.Getenv("BRAND_PERSONA"))
	if persona == "" {
		persona = brandPersonaForNiche(niche)
	}

	watermark := strings.TrimSpace(os.Getenv("BRAND_WATERMARK"))
	if watermark == "" {
		watermark = fmt.Sprintf("%s MODE", strings.ToUpper(slug))
	}

	introText := strings.TrimSpace(os.Getenv("BRAND_INTRO_TEXT"))
	if introText == "" {
		introText = fmt.Sprintf("%s presents", persona)
	}

	outroText := strings.TrimSpace(os.Getenv("BRAND_OUTRO_TEXT"))
	if outroText == "" {
		outroText = fmt.Sprintf("Follow for more %s", strings.TrimSpace(niche))
	}

	accentColor := strings.TrimSpace(os.Getenv("BRAND_ACCENT_COLOR"))
	if accentColor == "" {
		accentColor = "#3b82f6"
	}

	avatarPath := filepath.Join(s.assetDir, slug, "avatar.png")
	if _, err := os.Stat(avatarPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	profile := &domain.BrandProfile{
		Persona:     persona,
		Watermark:   watermark,
		IntroText:   introText,
		OutroText:   outroText,
		AvatarPath:  avatarPath,
		AccentColor: accentColor,
		AssetKey:    slug,
		Description: fmt.Sprintf("Brand profile for %s", niche),
	}

	if _, err := os.Stat(avatarPath); err == nil {
		return profile, nil
	}
	if s.image == nil {
		return profile, errors.New("image generator unavailable for avatar creation")
	}

	if err := os.MkdirAll(filepath.Dir(avatarPath), 0o755); err != nil {
		return profile, err
	}

	avatarPrompt := strings.TrimSpace(os.Getenv("BRAND_AVATAR_PROMPT"))
	if avatarPrompt == "" {
		avatarPrompt = fmt.Sprintf(
			"consistent AI avatar portrait for a %s content channel, %s persona, centered bust shot, clean gradient background, high detail, cinematic lighting, no text, no watermark, no extra faces",
			niche,
			persona,
		)
	}

	tmpPath, err := s.image.GenerateImage(ctx, avatarPrompt, brandAvatarSceneID(slug))
	if err != nil {
		return profile, err
	}

	if err := copyFile(tmpPath, avatarPath); err != nil {
		return profile, err
	}
	_ = os.Remove(tmpPath)
	return profile, nil
}

func brandSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '_' || r == '-' || r == '/':
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func brandPersonaForNiche(niche string) string {
	normalized := strings.ToLower(strings.TrimSpace(niche))
	switch {
	case strings.Contains(normalized, "stoic"):
		return "Stoic Sage"
	case strings.Contains(normalized, "finance"):
		return "Wealth Architect"
	case strings.Contains(normalized, "fitness"):
		return "Discipline Coach"
	case strings.Contains(normalized, "mind"):
		return "Mindset Mentor"
	default:
		title := strings.TrimSpace(niche)
		if title == "" {
			title = "Faceless Creator"
		}
		return strings.Title(title) + " Persona"
	}
}

func brandAvatarSceneID(slug string) int {
	sum := 0
	for _, r := range slug {
		sum += int(r)
	}
	if sum < 0 {
		return 9000
	}
	return 9000 + sum%1000
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

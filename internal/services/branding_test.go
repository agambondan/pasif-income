package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type stubBrandImageGenerator struct {
	calls int
	dir   string
}

func (s *stubBrandImageGenerator) GenerateImage(ctx context.Context, prompt string, sceneID int) (string, error) {
	s.calls++
	path := filepath.Join(s.dir, "generated-avatar.png")
	if err := os.WriteFile(path, []byte("avatar"), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func TestBrandingResolveCachesAvatar(t *testing.T) {
	dir := t.TempDir()
	gen := &stubBrandImageGenerator{dir: dir}
	t.Setenv("BRANDING_ASSET_DIR", filepath.Join(dir, "branding-assets"))

	svc := NewBrandingService(gen)
	profile, err := svc.Resolve(context.Background(), "stoicism")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile")
	}
	if profile.Persona == "" || profile.Watermark == "" || profile.AvatarPath == "" {
		t.Fatalf("expected populated profile: %#v", profile)
	}
	if _, err := os.Stat(profile.AvatarPath); err != nil {
		t.Fatalf("expected cached avatar file: %v", err)
	}
	if gen.calls != 1 {
		t.Fatalf("expected one image generation call, got %d", gen.calls)
	}

	again, err := svc.Resolve(context.Background(), "stoicism")
	if err != nil {
		t.Fatalf("resolve cached failed: %v", err)
	}
	if again == nil || again.AvatarPath != profile.AvatarPath {
		t.Fatalf("expected same avatar path, got %#v vs %#v", again, profile)
	}
	if gen.calls != 1 {
		t.Fatalf("expected cache hit without extra image generation, got %d calls", gen.calls)
	}
}

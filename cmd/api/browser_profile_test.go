package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserProfileStatus(t *testing.T) {
	t.Parallel()

	t.Run("missing", func(t *testing.T) {
		t.Parallel()
		if got := browserProfileStatus(filepath.Join(t.TempDir(), "missing")); got != "missing" {
			t.Fatalf("expected missing, got %q", got)
		}
	})

	t.Run("provisioned", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if got := browserProfileStatus(dir); got != "provisioned" {
			t.Fatalf("expected provisioned, got %q", got)
		}
	})

	t.Run("needs_login", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "profile.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("write profile marker: %v", err)
		}
		if got := browserProfileStatus(dir); got != "needs_login" {
			t.Fatalf("expected needs_login, got %q", got)
		}
	})

	t.Run("ready", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		defaultDir := filepath.Join(dir, "Default")
		if err := os.MkdirAll(defaultDir, 0o755); err != nil {
			t.Fatalf("mkdir default: %v", err)
		}
		if err := os.WriteFile(filepath.Join(defaultDir, "Cookies"), []byte("cookie"), 0o644); err != nil {
			t.Fatalf("write cookies: %v", err)
		}
		if got := browserProfileStatus(dir); got != "ready" {
			t.Fatalf("expected ready, got %q", got)
		}
	})
}

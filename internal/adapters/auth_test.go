package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCredentialDiscoveryFromLocalFiles(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_ACCESS_TOKEN", "")
	t.Setenv("GEMINI_ACCESS_TOKEN_FROM_OAUTH_CREDS", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_ACCESS_TOKEN", "")

	geminiDir := filepath.Join(tmpHome, ".gemini")
	codexDir := filepath.Join(tmpHome, ".codex")
	if err := os.MkdirAll(geminiDir, 0o755); err != nil {
		t.Fatalf("mkdir gemini: %v", err)
	}
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatalf("mkdir codex: %v", err)
	}

	if err := os.WriteFile(filepath.Join(geminiDir, "oauth_creds.json"), []byte(`{"access_token":"gemini-local-token","token_type":"Bearer"}`), 0o600); err != nil {
		t.Fatalf("write gemini creds: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "auth.json"), []byte(`{"tokens":{"access_token":"codex-local-token","account_id":"acct-1"}}`), 0o600); err != nil {
		t.Fatalf("write codex creds: %v", err)
	}

	if !HasGeminiCredentials() {
		t.Fatal("expected Gemini credentials to be discovered from ~/.gemini/oauth_creds.json")
	}
	if !HasCodexCredentials() {
		t.Fatal("expected Codex credentials to be discovered from ~/.codex/auth.json")
	}
}

func TestCredentialDiscoveryPrefersEnvOverLocalFiles(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("GEMINI_API_KEY", "gemini-env-key")
	t.Setenv("OPENAI_API_KEY", "openai-env-key")

	if err := os.MkdirAll(filepath.Join(tmpHome, ".gemini"), 0o755); err != nil {
		t.Fatalf("mkdir gemini: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpHome, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir codex: %v", err)
	}

	if !HasGeminiCredentials() {
		t.Fatal("expected Gemini env credentials to be accepted")
	}
	if !HasCodexCredentials() {
		t.Fatal("expected Codex env credentials to be accepted")
	}
}

package adapters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type codexAuthFile struct {
	OpenAIAPIKey string `json:"OPENAI_API_KEY"`
	Tokens       struct {
		AccessToken  string `json:"access_token"`
		AccountID    string `json:"account_id"`
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"tokens"`
}

type geminiOAuthFile struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func HasGeminiCredentials() bool {
	if strings.TrimSpace(os.Getenv("GEMINI_API_KEY")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("GEMINI_ACCESS_TOKEN")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("GEMINI_ACCESS_TOKEN_FROM_OAUTH_CREDS")) != "" {
		return true
	}
	return readGeminiOAuthToken() != ""
}

func HasCodexCredentials() bool {
	if strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("OPENAI_ACCESS_TOKEN")) != "" {
		return true
	}
	return readCodexAuthToken() != ""
}

func readGeminiOAuthToken() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}

	path := filepath.Join(home, ".gemini", "oauth_creds.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var creds geminiOAuthFile
	if err := json.Unmarshal(data, &creds); err != nil {
		return ""
	}

	return strings.TrimSpace(creds.AccessToken)
}

func readCodexAuthToken() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}

	path := filepath.Join(home, ".codex", "auth.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var auth codexAuthFile
	if err := json.Unmarshal(data, &auth); err != nil {
		return ""
	}

	if token := strings.TrimSpace(auth.Tokens.AccessToken); token != "" {
		return token
	}

	return strings.TrimSpace(auth.OpenAIAPIKey)
}

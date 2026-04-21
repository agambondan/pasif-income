package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BrowserLaunchRequest struct {
	ID          string    `json:"id"`
	UserID      int       `json:"user_id"`
	AccountID   string    `json:"account_id"`
	PlatformID  string    `json:"platform_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	ProfilePath string    `json:"profile_path"`
	TargetURL   string    `json:"target_url"`
	RequestedAt time.Time `json:"requested_at"`
	RequestedBy string    `json:"requested_by,omitempty"`
}

type BrowserLaunchQueue struct {
	dir string
}

func NewBrowserLaunchQueueFromEnv() *BrowserLaunchQueue {
	dir := strings.TrimSpace(os.Getenv("BROWSER_LAUNCH_REQUEST_DIR"))
	if dir == "" {
		dir = ".runtime/browser-launch-requests"
	}
	return &BrowserLaunchQueue{dir: dir}
}

func (q *BrowserLaunchQueue) Enqueue(ctx context.Context, req BrowserLaunchRequest) (string, error) {
	_ = ctx
	if q == nil {
		return "", fmt.Errorf("browser launch queue unavailable")
	}
	if strings.TrimSpace(req.ProfilePath) == "" {
		return "", fmt.Errorf("profile path is required")
	}
	if strings.TrimSpace(req.TargetURL) == "" {
		return "", fmt.Errorf("target url is required")
	}
	if req.RequestedAt.IsZero() {
		req.RequestedAt = time.Now().UTC()
	}
	if req.ID == "" {
		req.ID = fmt.Sprintf("%d", req.RequestedAt.UnixNano())
	}
	if err := os.MkdirAll(q.dir, 0o755); err != nil {
		return "", err
	}

	payload, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s_%s.json", req.ID, sanitizeArtifactName(req.AccountID))
	tmp, err := os.CreateTemp(q.dir, name+".tmp")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	finalPath := filepath.Join(q.dir, name)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return finalPath, nil
}

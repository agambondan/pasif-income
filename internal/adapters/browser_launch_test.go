package adapters

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserLaunchQueueEnqueue(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	queue := &BrowserLaunchQueue{dir: dir}
	path, err := queue.Enqueue(context.Background(), BrowserLaunchRequest{
		AccountID:   "youtube-agam",
		ProfilePath: "/tmp/profile",
		TargetURL:   "https://www.youtube.com/upload",
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected request file: %v", err)
	}
	if filepath.Dir(path) != dir {
		t.Fatalf("expected request file in %s, got %s", dir, path)
	}
}

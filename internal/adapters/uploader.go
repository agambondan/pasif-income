package adapters

import (
	"context"
	"log"
	"time"
)

type MockUploader struct {
	platform string
}

func NewMockUploader(platform string) *MockUploader {
	return &MockUploader{platform: platform}
}

func (u *MockUploader) Upload(ctx context.Context, filePath, title, description string) error {
	log.Printf("Starting upload to %s...\n", u.platform)
	log.Printf("File: %s\n", filePath)
	log.Printf("Title: %s\n", title)
	
	// Simulate upload time
	select {
	case <-time.After(2 * time.Second):
		log.Printf("Successfully uploaded to %s!\n", u.platform)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

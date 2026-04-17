package adapters

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/ports"
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

type MinIOUploader struct {
	storage  ports.Storage
	platform string
}

func NewMinIOUploader(endpoint, accessKey, secretKey, bucket, platform string) (*MinIOUploader, error) {
	storage, err := NewMinIOStorage(endpoint, accessKey, secretKey, bucket)
	if err != nil {
		return nil, err
	}

	if platform == "" {
		platform = "MinIO"
	}

	return &MinIOUploader{
		storage:  storage,
		platform: platform,
	}, nil
}

func (u *MinIOUploader) Upload(ctx context.Context, filePath, title, description string) error {
	objectName := fmt.Sprintf("%d_%s.mp4", time.Now().Unix(), sanitizeObjectName(title))
	log.Printf("Uploading final video to %s as %s...\n", u.platform, objectName)

	url, err := u.storage.Upload(ctx, filePath, objectName)
	if err != nil {
		return err
	}

	log.Printf("Uploaded to %s: %s\n", u.platform, url)
	return nil
}

func sanitizeObjectName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "faceless_video"
	}

	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '-':
			b.WriteRune('_')
		}
	}

	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "faceless_video"
	}
	return out
}

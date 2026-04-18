package adapters

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
	client     *minio.Client
	bucketName string
}

func NewMinIOStorage(endpoint, accessKey, secretKey, bucket string) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
		log.Printf("Created bucket: %s\n", bucket)
	}

	return &MinIOStorage{client, bucket}, nil
}

func (s *MinIOStorage) Upload(ctx context.Context, filePath, objectName string) (string, error) {
	_, err := s.client.FPutObject(ctx, s.bucketName, objectName, filePath, minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		return "", fmt.Errorf("minio put: %v", err)
	}

	// Generate a pre-signed URL for temporary access (1 day)
	reqParams := make(url.Values)
	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucketName, objectName, time.Hour*24, reqParams)
	if err != nil {
		return "", fmt.Errorf("minio presign: %v", err)
	}

	return presignedURL.String(), nil
}

func (s *MinIOStorage) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string
	objectCh := s.client.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		files = append(files, object.Key)
	}
	return files, nil
}

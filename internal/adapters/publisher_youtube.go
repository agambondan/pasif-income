package adapters

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	yt "google.golang.org/api/youtube/v3"
)

type YouTubeAPIUploadAdapter struct{}

func NewYouTubeAPIUploadAdapter() *YouTubeAPIUploadAdapter {
	return &YouTubeAPIUploadAdapter{}
}

func (a *YouTubeAPIUploadAdapter) Publish(ctx context.Context, filePath, title, description string, account domain.ConnectedAccount, progress func(string)) (string, error) {
	accessToken := strings.TrimSpace(account.AccessToken)
	if accessToken == "" {
		accessToken = strings.TrimSpace(os.Getenv("YOUTUBE_ACCESS_TOKEN"))
	}
	if accessToken == "" {
		return "", fmt.Errorf("missing youtube access token for account %s", account.ID)
	}

	log.Printf("Publishing %s to YouTube API for account %s\n", filePath, account.DisplayName)
	log.Printf("Title: %s\n", title)
	log.Printf("Description: %s\n", description)
	if progress != nil {
		progress("api_initializing")
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	svc, err := yt.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return "", err
	}
	if progress != nil {
		progress("api_uploading")
	}

	fh, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer fh.Close()

	privacyStatus := strings.TrimSpace(os.Getenv("YOUTUBE_PRIVACY_STATUS"))
	if privacyStatus == "" {
		privacyStatus = "private"
	}

	video := &yt.Video{
		Snippet: &yt.VideoSnippet{
			Title:       title,
			Description: description,
			CategoryId:  "22",
		},
		Status: &yt.VideoStatus{
			PrivacyStatus: privacyStatus,
		},
	}

	call := svc.Videos.Insert([]string{"snippet", "status"}, video)
	call.Media(fh)

	uploaded, err := call.Context(ctx).Do()
	if err != nil {
		return "", err
	}
	if uploaded == nil {
		return "", fmt.Errorf("youtube upload returned empty response")
	}
	if progress != nil {
		progress("api_completed")
	}

	return uploaded.Id, nil
}

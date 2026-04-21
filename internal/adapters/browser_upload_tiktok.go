package adapters

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func automateTikTokUpload(ctx context.Context, filePath, title, description string, progress func(string)) error {
	if progress != nil {
		progress("opening_upload_flow")
	}
	_ = clickFirstTextMatch(ctx, tiktokUploadActionTerms())

	if progress != nil {
		progress("attaching_file")
	}
	if err := waitAndSetUploadFile(ctx, "tiktok", filePath); err != nil {
		return err
	}

	if progress != nil {
		progress("waiting_for_processing")
	}
	// Wait for the "Edit" button or title field to become stable, indicating upload finished
	// TikTok usually shows a progress bar or "Uploading" text
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(tiktokCaptionSelectors()[0], chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil {
		log.Printf("Warning: waiting for tiktok processing timed out: %v\n", err)
	}

	if progress != nil {
		progress("filling_metadata")
	}
	caption := title
	if description != "" {
		caption = title + " " + description
	}
	_ = setFirstValue(ctx, tiktokCaptionSelectors(), caption)
	_ = waitShort(ctx)

	if progress != nil {
		progress("publishing")
	}
	if err := clickFirstTextMatch(ctx, tiktokPublishActionTerms()); err != nil {
		return fmt.Errorf("tiktok post action button not found: %w", err)
	}

	_ = waitShort(ctx)
	log.Println("TikTok publication initiated successfully")
	return nil
}

func tiktokUploadActionTerms() []string {
	return []string{"Upload", "Select file", "Choose file", "Post", "Import", "Add video"}
}

func tiktokPublishActionTerms() []string {
	return []string{"Post", "Publish", "Continue", "Upload"}
}

func tiktokCaptionSelectors() []string {
	return []string{
		"input[placeholder*='title']",
		"textarea[placeholder*='title']",
		"input[aria-label*='title']",
		"textarea[aria-label*='title']",
		"input[type='text']",
	}
}

package adapters

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func automateYouTubeUpload(ctx context.Context, filePath, title, description string, progress func(string)) error {
	if progress != nil {
		progress("attaching_file")
	}
	if err := waitAndSetUploadFile(ctx, "youtube", filePath); err != nil {
		return err
	}

	if progress != nil {
		progress("waiting_for_form")
	}
	if err := waitForAnyVisible(ctx, append(titleSelectors("youtube"), descriptionSelectors("youtube")...)); err != nil {
		return fmt.Errorf("youtube upload form not ready: %w", err)
	}

	if progress != nil {
		progress("filling_metadata")
	}
	if err := setFirstValue(ctx, titleSelectors("youtube"), title); err != nil {
		log.Printf("youtube title field not found: %v\n", err)
	}
	if err := setFirstValue(ctx, descriptionSelectors("youtube"), description); err != nil {
		log.Printf("youtube description field not found: %v\n", err)
	}

	if progress != nil {
		progress("setting_audience")
	}
	// YouTube requires "made for kids" selection
	_ = clickFirstTextMatch(ctx, []string{
		"no, it's not made for kids",
		"no, this is not made for kids",
	})
	
	_ = waitShort(ctx)

	if progress != nil {
		progress("advancing_to_video_elements")
	}
	if err := clickFirstTextMatch(ctx, []string{"Next"}); err != nil {
		return fmt.Errorf("youtube next button (1) not found: %w", err)
	}
	
	_ = waitShort(ctx)
	if err := clickFirstTextMatch(ctx, []string{"Next"}); err != nil {
		log.Printf("Warning: youtube next button (2) might have been skipped: %v\n", err)
	}

	if progress != nil {
		progress("waiting_for_checks")
	}
	_ = waitShort(ctx)
	_ = clickFirstTextMatch(ctx, []string{"Next"})

	if progress != nil {
		progress("setting_visibility")
	}
	
	_ = waitShort(ctx)
	
	privacy := strings.ToLower(strings.TrimSpace(os.Getenv("YOUTUBE_PRIVACY_STATUS")))
	if privacy == "" {
		privacy = "public"
	}
	
	log.Printf("Setting YouTube privacy to: %s\n", privacy)
	_ = clickFirstTextMatch(ctx, []string{privacy})

	if progress != nil {
		progress("waiting_for_upload_completion")
	}
	
	err := chromedp.Run(ctx, 
		chromedp.WaitVisible(`span:has-text("Upload complete")`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		log.Printf("Warning: timed out waiting for 'Upload complete' text, proceeding anyway: %v\n", err)
	}

	if progress != nil {
		progress("publishing")
	}
	
	if err := clickFirstTextMatch(ctx, []string{"Publish", "Save", "Done"}); err != nil {
		return fmt.Errorf("youtube final action button not found: %w", err)
	}

	_ = waitShort(ctx)
	log.Println("YouTube publication initiated successfully")
	return nil
}

func youtubeUploadActionTerms() []string {
	return uploadActionTerms("youtube")
}

func youtubePublishActionTerms() []string {
	return publishActionTerms("youtube")
}

func youtubeTitleSelectors() []string {
	return titleSelectors("youtube")
}

func youtubeDescriptionSelectors() []string {
	return descriptionSelectors("youtube")
}

package adapters

import (
	"context"
	"fmt"
	"log"
)

func automateInstagramUpload(ctx context.Context, filePath, title, description string, progress func(string)) error {
	if progress != nil {
		progress("opening_upload_flow")
	}
	if err := clickFirstTextMatch(ctx, instagramUploadActionTerms()); err != nil {
		return fmt.Errorf("instagram create button not found: %w", err)
	}

	// Wait for the modal/dropzone to be ready
	_ = waitShort(ctx)

	if progress != nil {
		progress("attaching_file")
	}
	if err := waitAndSetUploadFile(ctx, "instagram", filePath); err != nil {
		return err
	}

	_ = waitShort(ctx)

	if progress != nil {
		progress("adjusting_format")
	}
	if err := clickFirstTextMatch(ctx, []string{"Next"}); err != nil {
		return fmt.Errorf("instagram next (crop) button not found: %w", err)
	}
	_ = waitShort(ctx)
	if err := clickFirstTextMatch(ctx, []string{"Next"}); err != nil {
		return fmt.Errorf("instagram next (filter) button not found: %w", err)
	}

	if progress != nil {
		progress("filling_caption")
	}
	_ = waitShort(ctx)

	caption := title
	if description != "" {
		caption = title + "\n\n" + description
	}
	_ = setFirstValue(ctx, instagramCaptionSelectors(), caption)

	if progress != nil {
		progress("publishing")
	}
	if err := clickFirstTextMatch(ctx, instagramPublishActionTerms()); err != nil {
		return fmt.Errorf("instagram share button not found: %w", err)
	}

	_ = waitShort(ctx)
	log.Println("Instagram publication initiated successfully")
	return nil
}

func instagramUploadActionTerms() []string {
	return []string{"New post", "Select from computer", "Choose from computer", "Create new", "Create", "Next"}
}

func instagramPublishActionTerms() []string {
	return []string{"Share", "Publish", "Next", "Continue", "Share now"}
}

func instagramCaptionSelectors() []string {
	return []string{
		"textarea[placeholder*='Write a caption']",
		"textarea[placeholder*='caption']",
		"textarea[aria-label*='caption']",
		"textarea",
	}
}

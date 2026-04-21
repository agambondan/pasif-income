package adapters

import (
	"context"
	"fmt"
)

func automateGenericUpload(ctx context.Context, platformID, filePath, title, description string, progress func(string)) error {
	if progress != nil {
		progress("opening_upload_flow")
	}
	if err := clickFirstTextMatch(ctx, uploadActionTerms(platformID)); err != nil {
		return fmt.Errorf("upload action not found for %s: %w", platformID, err)
	}

	if progress != nil {
		progress("attaching_file")
	}
	if err := waitAndSetUploadFile(ctx, platformID, filePath); err != nil {
		return err
	}

	if progress != nil {
		progress("filling_metadata")
	}
	_ = setFirstValue(ctx, titleSelectors(platformID), title)
	_ = setFirstValue(ctx, descriptionSelectors(platformID), description)

	if progress != nil {
		progress("publishing")
	}
	if err := clickFirstTextMatch(ctx, publishActionTerms(platformID)); err != nil {
		return fmt.Errorf("publish action not found for %s: %w", platformID, err)
	}
	return nil
}

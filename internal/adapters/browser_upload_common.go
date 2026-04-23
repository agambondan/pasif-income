package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func BrowserTargetURL(platformID string) string {
	switch platformID {
	case "youtube":
		if url := strings.TrimSpace(os.Getenv("YOUTUBE_UPLOAD_URL")); url != "" {
			return url
		}
		return "https://www.youtube.com/upload"
	case "tiktok":
		if url := strings.TrimSpace(os.Getenv("TIKTOK_UPLOAD_URL")); url != "" {
			return url
		}
		return "https://www.tiktok.com/upload?lang=en"
	case "instagram":
		if url := strings.TrimSpace(os.Getenv("INSTAGRAM_UPLOAD_URL")); url != "" {
			return url
		}
		return "https://www.instagram.com/create/select/"
	default:
		return ""
	}
}

func browserTargetURL(platformID string) string {
	return BrowserTargetURL(platformID)
}

func waitAndSetUploadFile(ctx context.Context, platformID, filePath string) error {
	selectors := uploadFileSelectors(platformID)
	for _, selector := range selectors {
		if err := chromedp.Run(ctx,
			chromedp.WaitVisible(selector, chromedp.ByQuery),
			chromedp.SetUploadFiles(selector, []string{filePath}, chromedp.ByQuery),
		); err == nil {
			return nil
		}
	}

	var title string
	_ = chromedp.Run(ctx, chromedp.Title(&title))
	return fmt.Errorf("upload file input not found on page: %q", title)
}

func waitForAnyVisible(ctx context.Context, selectors []string) error {
	for _, selector := range selectors {
		if err := chromedp.Run(ctx, chromedp.WaitVisible(selector, chromedp.ByQuery)); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no matching visible element found")
}

func waitShort(ctx context.Context) error {
	timer := time.NewTicker(1500 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func waitForText(ctx context.Context, terms []string) error {
	if len(terms) == 0 {
		return fmt.Errorf("text terms are required")
	}

	normalized := make([]string, 0, len(terms))
	for _, term := range terms {
		if trimmed := strings.TrimSpace(strings.ToLower(term)); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) == 0 {
		return fmt.Errorf("text terms are required")
	}

	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}

	expr := fmt.Sprintf(`(() => {
		const terms = %s;
		const text = (document.body?.innerText || document.documentElement?.innerText || '').toLowerCase();
		return terms.some((term) => text.includes(term));
	})()`, payload)

	var matched bool
	return chromedp.Run(ctx,
		chromedp.Poll(expr, &matched,
			chromedp.WithPollingInterval(500*time.Millisecond),
			chromedp.WithPollingTimeout(browserAutomationTimeout()),
		),
	)
}

func setFirstValue(ctx context.Context, selectors []string, value string) error {
	for _, selector := range selectors {
		var present bool
		presenceJS := fmt.Sprintf(`(() => document.querySelector(%q) !== null)()`, selector)
		if err := chromedp.Run(ctx, chromedp.Evaluate(presenceJS, &present)); err != nil || !present {
			continue
		}

		if err := chromedp.Run(ctx,
			chromedp.SetValue(selector, value, chromedp.ByQuery),
			chromedp.Blur(selector, chromedp.ByQuery),
		); err == nil {
			return nil
		}

		js := fmt.Sprintf(`(() => {
			const el = document.querySelector(%q);
			if (!el) {
				return false;
			}
			el.focus?.();
			el.value = %q;
			el.dispatchEvent(new Event('input', { bubbles: true }));
			el.dispatchEvent(new Event('change', { bubbles: true }));
			el.blur?.();
			return true;
		})()`, selector, value)
		var applied bool
		if err := chromedp.Run(ctx, chromedp.Evaluate(js, &applied)); err == nil && applied {
			return nil
		}
	}
	return fmt.Errorf("no matching field found")
}

func clickFirstTextMatch(ctx context.Context, terms []string) error {
	for _, term := range terms {
		js := fmt.Sprintf(`(() => {
			const needle = %q;
			const all = Array.from(document.querySelectorAll('*'));
			const clickable = all.filter((el) => {
				const tag = (el.tagName || '').toUpperCase();
				return tag === 'BUTTON' ||
					tag === 'A' ||
					tag === 'TP-YT-PAPER-BUTTON' ||
					tag === 'YTCP-BUTTON' ||
					tag === 'INPUT' ||
					el.getAttribute('role') === 'button';
			});
			const nodes = clickable.length > 0 ? clickable : all;
			const match = nodes.find((el) => {
				const candidates = [
					el.innerText,
					el.textContent,
					el.value,
					el.getAttribute('aria-label'),
					el.getAttribute('title'),
				].filter(Boolean).map((value) => value.trim().toLowerCase());
				return candidates.some((value) => value.includes(needle));
			});
			if (match) {
				match.click();
				return true;
			}
			return false;
		})()`, strings.ToLower(term))
		var clicked bool
		if err := chromedp.Run(ctx, chromedp.Evaluate(js, &clicked)); err == nil && clicked {
			return nil
		}
	}

	var visibleTexts []string
	diagJS := `(() => {
		const nodes = Array.from(document.querySelectorAll('*'));
		return nodes
			.map((el) => [
				el.tagName,
				el.id || '',
				el.getAttribute('aria-label') || '',
				el.getAttribute('title') || '',
				(el.innerText || el.textContent || el.value || '').trim(),
			].filter(Boolean).join(' | '))
			.filter((value) => value.length > 0)
			.slice(0, 30);
	})()`
	if err := chromedp.Run(ctx, chromedp.Evaluate(diagJS, &visibleTexts)); err != nil {
		return fmt.Errorf("no matching text action found")
	}
	return fmt.Errorf("no matching text action found; visible=%v", visibleTexts)
}

func uploadActionTerms(platformID string) []string {
	switch platformID {
	case "youtube":
		return []string{"Create", "Upload videos", "Select files", "Upload"}
	case "tiktok":
		return []string{"Upload", "Select file", "Post", "Import"}
	case "instagram":
		return []string{"New post", "Select from computer", "Create", "Next"}
	default:
		return []string{"Upload", "Select file"}
	}
}

func publishActionTerms(platformID string) []string {
	switch platformID {
	case "youtube":
		return []string{"Publish", "Done", "Next", "Save"}
	case "tiktok":
		return []string{"Post", "Publish", "Continue"}
	case "instagram":
		return []string{"Share", "Publish", "Next", "Share now"}
	default:
		return []string{"Publish", "Share", "Post"}
	}
}

func titleSelectors(platformID string) []string {
	switch platformID {
	case "youtube":
		return []string{"textarea#title-textarea", "input#title-textarea", "input[aria-label*='title']"}
	case "tiktok":
		return []string{"input[placeholder*='title']", "textarea[placeholder*='title']"}
	case "instagram":
		return []string{"textarea[placeholder*='Write a caption']", "textarea"}
	default:
		return []string{"input[aria-label*='title']", "textarea"}
	}
}

func descriptionSelectors(platformID string) []string {
	switch platformID {
	case "youtube":
		return []string{"ytcp-social-suggestions-textbox#description-textarea", "textarea#description-textarea"}
	case "tiktok":
		return []string{"textarea[placeholder*='description']"}
	case "instagram":
		return []string{"textarea[placeholder*='caption']"}
	default:
		return []string{"textarea"}
	}
}

func uploadFileSelectors(platformID string) []string {
	base := []string{"input[type=file]", "input[name=file]"}
	switch platformID {
	case "youtube":
		return append([]string{"input[accept*='video/*']"}, base...)
	case "tiktok":
		return append([]string{"input[accept*='video']"}, base...)
	case "instagram":
		return append([]string{"input[accept*='video']", "input[accept*='video/*']", "input[accept*='image']"}, base...)
	default:
		return base
	}
}

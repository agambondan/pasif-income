package services

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

type AffiliateService struct {
	enabled       bool
	baseURL       string
	disclosure    string
	defaultOffers []domain.AffiliateProduct
}

func NewAffiliateService() *AffiliateService {
	enabled := true
	if raw := strings.TrimSpace(os.Getenv("AFFILIATE_ENABLED")); raw != "" {
		switch strings.ToLower(raw) {
		case "0", "false", "no", "off":
			enabled = false
		}
	}

	baseURL := strings.TrimSpace(os.Getenv("AFFILIATE_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://example.com/affiliate"
	}

	disclosure := strings.TrimSpace(os.Getenv("AFFILIATE_DISCLOSURE"))
	if disclosure == "" {
		disclosure = "Disclosure: this post contains affiliate links."
	}

	return &AffiliateService{
		enabled:       enabled,
		baseURL:       strings.TrimRight(baseURL, "/"),
		disclosure:    disclosure,
		defaultOffers: defaultAffiliateOffers(),
	}
}

func (s *AffiliateService) Build(niche, topic string) *domain.AffiliatePlan {
	if s == nil || !s.enabled {
		return nil
	}

	catalog := s.loadCatalog()
	if len(catalog) == 0 {
		catalog = s.defaultOffers
	}
	if len(catalog) == 0 {
		return nil
	}

	best, score, source := s.pickOffer(niche, topic, catalog)
	if score <= 0 && len(catalog) == 0 {
		return nil
	}
	if best.URL == "" {
		best.URL = s.genericURLFor(best.Title, niche)
	}
	if best.Disclosure == "" {
		best.Disclosure = s.disclosure
	}
	if best.CTA == "" {
		best.CTA = fmt.Sprintf("Check the recommended resource for %s.", normalizeAffiliateText(niche))
	}
	if best.PinComment == "" {
		best.PinComment = fmt.Sprintf("%s %s", best.CTA, best.URL)
	}

	description := s.composeDescription(niche, topic, best)
	return &domain.AffiliatePlan{
		Product:     best,
		MatchSource: source,
		Score:       score,
		Description: description,
		PinComment:  best.PinComment,
	}
}

func (s *AffiliateService) loadCatalog() []domain.AffiliateProduct {
	raw := strings.TrimSpace(os.Getenv("AFFILIATE_CATALOG_JSON"))
	if raw == "" {
		return nil
	}

	var catalog []domain.AffiliateProduct
	if err := json.Unmarshal([]byte(raw), &catalog); err != nil {
		return nil
	}
	return catalog
}

func (s *AffiliateService) pickOffer(niche, topic string, catalog []domain.AffiliateProduct) (domain.AffiliateProduct, int, string) {
	nicheNorm := strings.ToLower(normalizeAffiliateText(niche))
	topicNorm := strings.ToLower(normalizeAffiliateText(topic))

	bestScore := -1
	bestIndex := 0
	bestSource := "default-catalog"

	for i, offer := range catalog {
		score := scoreAffiliateOffer(offer, nicheNorm, topicNorm)
		if score > bestScore {
			bestScore = score
			bestIndex = i
			bestSource = offer.Title
		}
	}

	return catalog[bestIndex], bestScore, bestSource
}

func scoreAffiliateOffer(offer domain.AffiliateProduct, nicheNorm, topicNorm string) int {
	score := 0
	haystack := strings.ToLower(strings.Join(append([]string{
		offer.Title,
		offer.URL,
		offer.CTA,
		offer.Disclosure,
	}, offer.Tags...), " "))

	for _, token := range strings.Fields(nicheNorm + " " + topicNorm) {
		if len(token) < 3 {
			continue
		}
		if strings.Contains(haystack, token) {
			score += 10
		}
	}

	for _, tag := range offer.Tags {
		if strings.Contains(nicheNorm, strings.ToLower(tag)) || strings.Contains(topicNorm, strings.ToLower(tag)) {
			score += 8
		}
	}

	if strings.Contains(haystack, nicheNorm) {
		score += 20
	}
	if strings.Contains(haystack, topicNorm) {
		score += 16
	}
	return score
}

func (s *AffiliateService) composeDescription(niche, topic string, offer domain.AffiliateProduct) string {
	base := fmt.Sprintf("#%s #%s #ai #faceless", strings.ReplaceAll(strings.ToLower(strings.TrimSpace(niche)), " ", ""), strings.ReplaceAll(strings.ToLower(strings.TrimSpace(topic)), " ", ""))
	lines := []string{
		base,
		"",
		s.disclosure,
		fmt.Sprintf("Recommended: %s", offer.Title),
		fmt.Sprintf("Link: %s", offer.URL),
		fmt.Sprintf("CTA: %s", offer.CTA),
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (s *AffiliateService) genericURLFor(title, niche string) string {
	slug := slugifyAffiliate(strings.TrimSpace(title))
	if slug == "" {
		slug = slugifyAffiliate(niche)
	}
	if slug == "" {
		slug = "resource"
	}
	return fmt.Sprintf("%s/%s", s.baseURL, slug)
}

func defaultAffiliateOffers() []domain.AffiliateProduct {
	return []domain.AffiliateProduct{
		{
			Title:      "The Daily Stoic",
			URL:        "https://example.com/affiliate/daily-stoic",
			CTA:        "Read it before your next scroll session.",
			Disclosure: "Disclosure: this post contains affiliate links.",
			PinComment: "If this helped, grab the book here:",
			Tags:       []string{"stoicism", "discipline", "mindset"},
		},
		{
			Title:      "Deep Work",
			URL:        "https://example.com/affiliate/deep-work",
			CTA:        "Use this framework to stay locked in.",
			Disclosure: "Disclosure: this post contains affiliate links.",
			PinComment: "For the full system, check the link below:",
			Tags:       []string{"productivity", "focus", "deep work", "mindset"},
		},
		{
			Title:      "Atomic Habits",
			URL:        "https://example.com/affiliate/atomic-habits",
			CTA:        "Apply the habit loop from today.",
			Disclosure: "Disclosure: this post contains affiliate links.",
			PinComment: "This is the habit book I keep recommending:",
			Tags:       []string{"habits", "self improvement", "discipline", "productivity"},
		},
		{
			Title:      "Budget Planner",
			URL:        "https://example.com/affiliate/budget-planner",
			CTA:        "Start tracking money with a simple system.",
			Disclosure: "Disclosure: this post contains affiliate links.",
			PinComment: "For the money tracker I use, see the link:",
			Tags:       []string{"finance", "money", "budget", "wealth"},
		},
	}
}

func normalizeAffiliateText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_' || r == '/':
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func slugifyAffiliate(value string) string {
	value = normalizeAffiliateText(value)
	if value == "" {
		return ""
	}
	return strings.ReplaceAll(value, " ", "-")
}

func (s *AffiliateService) sortedOffers(niche, topic string, catalog []domain.AffiliateProduct) []domain.AffiliateProduct {
	typed := make([]struct {
		offer domain.AffiliateProduct
		score int
	}, 0, len(catalog))
	for _, offer := range catalog {
		typed = append(typed, struct {
			offer domain.AffiliateProduct
			score int
		}{offer: offer, score: scoreAffiliateOffer(offer, strings.ToLower(normalizeAffiliateText(niche)), strings.ToLower(normalizeAffiliateText(topic)))})
	}
	sort.SliceStable(typed, func(i, j int) bool {
		if typed[i].score == typed[j].score {
			return typed[i].offer.Title < typed[j].offer.Title
		}
		return typed[i].score > typed[j].score
	})
	out := make([]domain.AffiliateProduct, 0, len(typed))
	for _, item := range typed {
		out = append(out, item.offer)
	}
	return out
}

package services

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

type TrendResearchService struct {
	client *http.Client
}

func NewTrendResearchService() *TrendResearchService {
	timeout := 12 * time.Second
	if raw := strings.TrimSpace(os.Getenv("TREND_RESEARCH_TIMEOUT_SECONDS")); raw != "" {
		if secs, err := strconv.Atoi(raw); err == nil && secs >= 3 && secs <= 60 {
			timeout = time.Duration(secs) * time.Second
		}
	}
	return &TrendResearchService{
		client: &http.Client{Timeout: timeout},
	}
}

func (s *TrendResearchService) Research(ctx context.Context, niche string, limit int) (*domain.TrendResearchResult, error) {
	if s == nil {
		return nil, fmt.Errorf("trend research service unavailable")
	}
	if limit < 1 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	niche = strings.TrimSpace(niche)
	if niche == "" {
		niche = "stoicism"
	}

	accumulated := make(map[string]*trendAccumulator)
	var warnings []string

	if signals, err := s.fetchGoogleTrends(ctx); err != nil {
		warnings = append(warnings, err.Error())
	} else {
		for _, signal := range signals {
			appendTrendSignal(accumulated, signal)
		}
	}

	for _, seed := range trendSeedsForNiche(niche) {
		if signals, err := s.fetchAutocompleteSuggestions(ctx, seed, "yt"); err != nil {
			warnings = append(warnings, err.Error())
		} else {
			for _, signal := range signals {
				appendTrendSignal(accumulated, signal)
			}
		}

		if signals, err := s.fetchAutocompleteSuggestions(ctx, seed, "web"); err != nil {
			warnings = append(warnings, err.Error())
		} else {
			for _, signal := range signals {
				appendTrendSignal(accumulated, signal)
			}
		}
	}

	if len(accumulated) == 0 {
		for _, signal := range fallbackTrendSignals(niche) {
			appendTrendSignal(accumulated, signal)
		}
		warnings = append(warnings, "using local fallback trend signals")
	}

	signals := make([]domain.TrendSignal, 0, len(accumulated))
	for _, acc := range accumulated {
		signals = append(signals, acc.toSignal())
	}
	sort.SliceStable(signals, func(i, j int) bool {
		if signals[i].Score == signals[j].Score {
			return signals[i].Query < signals[j].Query
		}
		return signals[i].Score > signals[j].Score
	})

	if len(signals) > limit*4 {
		signals = signals[:limit*4]
	}

	ideas := synthesizeIdeas(niche, signals, limit)
	return &domain.TrendResearchResult{
		Niche:       niche,
		Seed:        niche,
		Signals:     signals,
		Ideas:       ideas,
		Warnings:    warnings,
		CollectedAt: time.Now().UTC(),
	}, nil
}

func (s *TrendResearchService) fetchGoogleTrends(ctx context.Context) ([]domain.TrendSignal, error) {
	endpoint := strings.TrimSpace(os.Getenv("GOOGLE_TRENDS_RSS_URL"))
	if endpoint == "" {
		endpoint = "https://trends.google.com/trends/trendingsearches/daily/rss?geo=US"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("trends rss failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Description string `xml:"description"`
				Link        string `xml:"link"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	signals := make([]domain.TrendSignal, 0, len(payload.Channel.Items))
	for idx, item := range payload.Channel.Items {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		signals = append(signals, domain.TrendSignal{
			Query:   title,
			Source:  "google-trends",
			Score:   100 - idx*3,
			Link:    strings.TrimSpace(item.Link),
			Context: truncateText(item.Description, 160),
		})
	}
	return signals, nil
}

func (s *TrendResearchService) fetchAutocompleteSuggestions(ctx context.Context, seed string, source string) ([]domain.TrendSignal, error) {
	seed = strings.TrimSpace(seed)
	if seed == "" {
		return nil, nil
	}

	ds := "web"
	scoreBase := 70
	if source == "yt" {
		ds = "yt"
		scoreBase = 80
	}

	endpoint := "https://suggestqueries.google.com/complete/search"
	params := url.Values{}
	params.Set("client", "firefox")
	params.Set("q", seed)
	params.Set("ds", ds)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s suggestions failed: %s: %s", source, resp.Status, strings.TrimSpace(string(body)))
	}

	var payload []any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload) < 2 {
		return nil, nil
	}

	rawList, ok := payload[1].([]any)
	if !ok {
		return nil, nil
	}

	signals := make([]domain.TrendSignal, 0, len(rawList))
	for idx, raw := range rawList {
		query, ok := raw.(string)
		if !ok || strings.TrimSpace(query) == "" {
			continue
		}
		signals = append(signals, domain.TrendSignal{
			Query:   query,
			Source:  "autocomplete-" + source,
			Score:   scoreBase - idx*2,
			Context: "autocomplete suggestion for " + seed,
		})
	}
	return signals, nil
}

type trendAccumulator struct {
	Query   string
	Score   int
	Sources map[string]int
	Link    string
	Context string
}

func appendTrendSignal(acc map[string]*trendAccumulator, signal domain.TrendSignal) {
	key := normalizeTrendKey(signal.Query)
	if key == "" {
		return
	}
	item, ok := acc[key]
	if !ok {
		item = &trendAccumulator{
			Query:   signal.Query,
			Score:   signal.Score,
			Sources: map[string]int{},
			Link:    signal.Link,
			Context: signal.Context,
		}
		acc[key] = item
	}
	if signal.Score > item.Score {
		item.Score = signal.Score
	}
	item.Sources[signal.Source]++
	if item.Link == "" && signal.Link != "" {
		item.Link = signal.Link
	}
	if item.Context == "" && signal.Context != "" {
		item.Context = signal.Context
	}
}

func (a *trendAccumulator) toSignal() domain.TrendSignal {
	sources := make([]string, 0, len(a.Sources))
	for source := range a.Sources {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	return domain.TrendSignal{
		Query:   a.Query,
		Source:  strings.Join(sources, ","),
		Score:   a.Score + len(sources)*4,
		Link:    a.Link,
		Context: a.Context,
	}
}

func synthesizeIdeas(niche string, signals []domain.TrendSignal, limit int) []domain.IdeaSuggestion {
	if limit < 1 {
		limit = 5
	}
	ideas := make([]domain.IdeaSuggestion, 0, limit)
	for _, signal := range signals {
		if len(ideas) >= limit {
			break
		}

		keyword := normalizeHumanText(signal.Query)
		nicheLabel := normalizeHumanText(niche)
		title := buildIdeaTitle(nicheLabel, keyword)
		hook := buildIdeaHook(nicheLabel, keyword)
		angle := buildIdeaAngle(nicheLabel, keyword, signal.Source)
		reason := signal.Context
		if reason == "" {
			reason = fmt.Sprintf("Signal from %s with score %d", signal.Source, signal.Score)
		}

		ideas = append(ideas, domain.IdeaSuggestion{
			Title:       title,
			Hook:        hook,
			Angle:       angle,
			SearchQuery: signal.Query,
			TrendSource: signal.Source,
			Score:       signal.Score,
			Reason:      reason,
		})
	}

	if len(ideas) == 0 {
		fallback := fallbackTrendSignals(niche)
		for _, signal := range fallback {
			if len(ideas) >= limit {
				break
			}
			keyword := normalizeHumanText(signal.Query)
			nicheLabel := normalizeHumanText(niche)
			ideas = append(ideas, domain.IdeaSuggestion{
				Title:       buildIdeaTitle(nicheLabel, keyword),
				Hook:        buildIdeaHook(nicheLabel, keyword),
				Angle:       buildIdeaAngle(nicheLabel, keyword, signal.Source),
				SearchQuery: signal.Query,
				TrendSource: signal.Source,
				Score:       signal.Score,
				Reason:      signal.Context,
			})
		}
	}

	return ideas
}

func buildIdeaTitle(niche string, keyword string) string {
	switch {
	case strings.HasPrefix(strings.ToLower(keyword), "how to "):
		return titleCase(keyword)
	case strings.HasPrefix(strings.ToLower(keyword), "why "):
		return fmt.Sprintf("Why %s for %s", strings.TrimSpace(keyword[4:]), niche)
	default:
		return fmt.Sprintf("%s trend for %s", titleCase(keyword), niche)
	}
}

func buildIdeaHook(niche string, keyword string) string {
	return fmt.Sprintf("Use %s as a hook to frame a %s video people actually stop for.", keyword, niche)
}

func buildIdeaAngle(niche string, keyword string, source string) string {
	return fmt.Sprintf("Blend the %s trend into a %s angle and publish it while the %s signal is still warm.", keyword, niche, source)
}

func fallbackTrendSignals(niche string) []domain.TrendSignal {
	base := []string{
		"how to " + niche + " fast",
		"best " + niche + " tips",
		niche + " mistakes to avoid",
		niche + " for beginners",
		"viral " + niche + " hook",
	}

	signals := make([]domain.TrendSignal, 0, len(base))
	for i, query := range base {
		signals = append(signals, domain.TrendSignal{
			Query:   query,
			Source:  "local-template",
			Score:   60 - i*3,
			Context: "generated from niche template",
		})
	}
	return signals
}

func trendSeedsForNiche(niche string) []string {
	niche = strings.TrimSpace(niche)
	if niche == "" {
		return []string{"trending", "viral shorts"}
	}

	seeds := []string{
		niche,
		niche + " tips",
		niche + " mistakes",
		niche + " shorts",
		niche + " viral",
		"how to " + niche,
	}
	if len(strings.Fields(niche)) > 1 {
		seeds = append(seeds, strings.ReplaceAll(niche, " ", ""))
	}
	return seeds
}

func normalizeTrendKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_' || r == '/':
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func normalizeHumanText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	parts := strings.Fields(strings.ToLower(value))
	for i, part := range parts {
		switch part {
		case "ai", "seo", "ugc", "usa":
			parts[i] = strings.ToUpper(part)
		default:
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

func titleCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	parts := strings.Fields(strings.ToLower(value))
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		if len(part) == 1 {
			parts[i] = strings.ToUpper(part)
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	if limit < 1 {
		return ""
	}
	return strings.TrimSpace(value[:limit]) + "..."
}

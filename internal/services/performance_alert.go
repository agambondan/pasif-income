package services

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

type PerformanceAlertService struct {
	dropThreshold float64
	minBaseline   uint64
}

func NewPerformanceAlertService() *PerformanceAlertService {
	return &PerformanceAlertService{
		dropThreshold: performanceDropThresholdFromEnv(),
		minBaseline:   performanceDropBaselineFromEnv(),
	}
}

func (s *PerformanceAlertService) Assess(history []domain.VideoMetricSnapshot) []domain.PerformanceAlert {
	if s == nil || len(history) == 0 {
		return nil
	}

	grouped := make(map[string][]domain.VideoMetricSnapshot)
	for _, snap := range history {
		key := strings.TrimSpace(snap.ExternalID)
		if key == "" {
			continue
		}
		grouped[key] = append(grouped[key], snap)
	}

	alerts := make([]domain.PerformanceAlert, 0)
	for externalID, snaps := range grouped {
		sort.Slice(snaps, func(i, j int) bool {
			return snaps[i].CollectedAt.After(snaps[j].CollectedAt)
		})
		if len(snaps) < 2 {
			continue
		}

		latest := snaps[0]
		previous := snaps[1]
		if previous.ViewCount < s.minBaseline {
			continue
		}
		if previous.ViewCount == 0 || latest.ViewCount >= previous.ViewCount {
			continue
		}

		drop := 100 * (1 - float64(latest.ViewCount)/float64(previous.ViewCount))
		if drop < s.dropThreshold {
			continue
		}

		alerts = append(alerts, domain.PerformanceAlert{
			ID:            fmt.Sprintf("%s-%s-%d", externalID, latest.Platform, latest.CollectedAt.Unix()),
			Level:         alertLevel(drop),
			Platform:      strings.TrimSpace(latest.Platform),
			AccountID:     strings.TrimSpace(latest.AccountID),
			Niche:         strings.TrimSpace(latest.Niche),
			ExternalID:    externalID,
			VideoTitle:    strings.TrimSpace(latest.VideoTitle),
			Metric:        "views",
			CurrentValue:  latest.ViewCount,
			PreviousValue: previous.ViewCount,
			DropPercent:   math.Round(drop*10) / 10,
			Message:       fmt.Sprintf("Views dropped %.1f%% from %d to %d for %s", drop, previous.ViewCount, latest.ViewCount, firstNonEmpty(strings.TrimSpace(latest.VideoTitle), externalID)),
			CreatedAt:     latest.CollectedAt,
		})
	}

	sort.Slice(alerts, func(i, j int) bool {
		if alerts[i].DropPercent == alerts[j].DropPercent {
			return alerts[i].CreatedAt.After(alerts[j].CreatedAt)
		}
		return alerts[i].DropPercent > alerts[j].DropPercent
	})
	return alerts
}

func alertLevel(drop float64) string {
	switch {
	case drop >= 60:
		return "critical"
	case drop >= 40:
		return "high"
	default:
		return "medium"
	}
}

func performanceDropThresholdFromEnv() float64 {
	value := parseFloatEnv("PERFORMANCE_DROP_THRESHOLD_PERCENT", 30)
	if value < 10 {
		return 10
	}
	if value > 90 {
		return 90
	}
	return value
}

func performanceDropBaselineFromEnv() uint64 {
	value := parseIntEnv("PERFORMANCE_MIN_BASE_VIEWS", 500)
	if value < 10 {
		return 10
	}
	return uint64(value)
}

func parseFloatEnv(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return value
}

func parseIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

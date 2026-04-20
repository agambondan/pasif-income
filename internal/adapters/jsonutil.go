package adapters

import (
	"encoding/json"
	"strings"
)

func extractJSONPayload(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}

	candidates := []struct {
		open  string
		close string
	}{
		{open: "{", close: "}"},
		{open: "[", close: "]"},
	}

	for _, candidate := range candidates {
		start := strings.Index(raw, candidate.open)
		end := strings.LastIndex(raw, candidate.close)
		if start < 0 || end <= start {
			continue
		}
		payload := strings.TrimSpace(raw[start : end+1])
		if json.Valid([]byte(payload)) {
			return payload
		}
	}

	return raw
}

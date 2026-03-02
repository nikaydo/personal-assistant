package ai

import (
	"encoding/json"
	"fmt"

	mod "github.com/nikaydo/personal-assistant/internal/models"
)

func parseToolArguments(raw string, out any) error {
	if err := json.Unmarshal([]byte(raw), out); err == nil {
		return nil
	} else {
		extracted, extractErr := mod.ExtractJSON(raw)
		if extractErr != nil {
			return fmt.Errorf("tool args parse failed: %w (raw: %s)", err, shortLogValue(raw, 240))
		}

		if err2 := json.Unmarshal([]byte(extracted), out); err2 != nil {
			return fmt.Errorf("tool args parse failed: first=%v, fallback=%v (raw: %s)", err, err2, shortLogValue(raw, 240))
		}
	}

	return nil
}

func shortLogValue(v string, max int) string {
	if max <= 3 || len(v) <= max {
		return v
	}
	return v[:max-3] + "..."
}

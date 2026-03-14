package llmcalls

import (
	"strings"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/models"
)

const webSearchPluginID = "web"

func ApplyWebSearch(body *models.RequestBody, cfg config.Config) {
	if body == nil {
		return
	}

	if !hasWebSearchPlugin(body.Plugins) {
		body.Plugins = append(body.Plugins, models.Plugin{ID: webSearchPluginID})
	}

	if size := normalizeWebSearchContextSize(cfg.LLMWebSearchContextSize); size != "" {
		if body.WebSearchOptions == nil {
			body.WebSearchOptions = &models.WebSearchOptions{}
		}
		body.WebSearchOptions.SearchContextSize = size
	}
}

func hasWebSearchPlugin(plugins []models.Plugin) bool {
	for _, plugin := range plugins {
		if strings.EqualFold(strings.TrimSpace(plugin.ID), webSearchPluginID) {
			return true
		}
	}
	return false
}

func normalizeWebSearchContextSize(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

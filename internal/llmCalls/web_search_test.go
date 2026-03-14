package llmcalls

import (
	"testing"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/models"
)

func TestApplyWebSearch_AddsPluginByDefault(t *testing.T) {
	body := models.RequestBody{}

	ApplyWebSearch(&body, config.Config{})

	if len(body.Plugins) != 1 || body.Plugins[0].ID != webSearchPluginID {
		t.Fatalf("unexpected plugins: %+v", body.Plugins)
	}
	if body.WebSearchOptions != nil {
		t.Fatalf("expected no extra options by default, got %+v", body.WebSearchOptions)
	}
}

func TestApplyWebSearch_AddsOptionsWhenConfigured(t *testing.T) {
	body := models.RequestBody{}

	ApplyWebSearch(&body, config.Config{LLMWebSearchContextSize: " HIGH "})

	if len(body.Plugins) != 1 || body.Plugins[0].ID != webSearchPluginID {
		t.Fatalf("unexpected plugins: %+v", body.Plugins)
	}
	if body.WebSearchOptions == nil {
		t.Fatal("expected web search options to be set")
	}
	if body.WebSearchOptions.SearchContextSize != "high" {
		t.Fatalf("unexpected search context size: %q", body.WebSearchOptions.SearchContextSize)
	}
}

func TestApplyWebSearch_DoesNotDuplicatePlugin(t *testing.T) {
	body := models.RequestBody{
		Plugins: []models.Plugin{{ID: webSearchPluginID}},
	}

	ApplyWebSearch(&body, config.Config{})

	if len(body.Plugins) != 1 {
		t.Fatalf("expected one plugin, got %+v", body.Plugins)
	}
}

func TestApplyWebSearch_IgnoresInvalidContextSize(t *testing.T) {
	body := models.RequestBody{}

	ApplyWebSearch(&body, config.Config{LLMWebSearchContextSize: "deep"})

	if len(body.Plugins) != 1 || body.Plugins[0].ID != webSearchPluginID {
		t.Fatalf("unexpected plugins: %+v", body.Plugins)
	}
	if body.WebSearchOptions != nil {
		t.Fatalf("expected invalid context size to be ignored, got %+v", body.WebSearchOptions)
	}
}

package tools

import (
	"strings"
	"testing"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/models"
)

func TestDetectChosenTool_ReturnsErrorOnEmptyEmbedding(t *testing.T) {
	oldCreateEmbedding := createEmbeddingFn
	createEmbeddingFn = func(input string, cfg config.Config) (models.EmbendingResponse, error) {
		return models.EmbendingResponse{}, nil
	}
	t.Cleanup(func() {
		createEmbeddingFn = oldCreateEmbedding
	})

	tool := &Tool{}
	err := tool.DetectChosenTool(models.ResponseBody{
		Choices: []models.Choices{
			{
				Message: models.Message{
					ToolCalls: []models.ToolCall{
						{
							Type: "function",
							Function: models.ToolFunction{
								Name: "summarize",
								Arguments: `{
									"text":"x",
									"category":"note",
									"goal":"save",
									"importance":"high",
									"status":"completed"
								}`,
							},
						},
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error for empty embedding response")
	}
	if !strings.Contains(err.Error(), "empty embedding response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

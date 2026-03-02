package memory

import "github.com/nikaydo/personal-assistant/internal/models"

func GetToolLongTerm() []models.Tool {
	return []models.Tool{
		{
			Type: "function",
			Function: models.Function{
				Name:        "summarize",
				Description: "A function for creating a short summary of a text based on its category, purpose, and importance.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"text": map[string]any{
							"type":        "string",
							"description": "Summarized text according to the history of previous queries",
						},
						"category": map[string]any{
							"type":        "string",
							"description": "Text category, such as 'news', 'report', 'technical document'. Used for contextual processing.",
						},
						"goal": map[string]any{
							"type":        "string",
							"description": "The purpose of the summary: for example, 'quick overview', 'report preparation', 'presentation creation'.",
						},
						"importance": map[string]any{
							"type":        "string",
							"description": "The importance level of the summary: 'high', 'medium', 'low'. Used to prioritize key information.",
						},
						"status": map[string]any{
							"type":        "string",
							"description": "The status of the summary: 'completed', 'in progress', 'failed'.",
						},
					},
					"required": []string{"text", "category", "goal", "importance", "status"},
				},
			},
		},
	}
}

package tools

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
		{
			Type: "function",
			Function: models.Function{
				Name:        "change_agent_settings",
				Description: "A function for creating a short summary of a text based on its category, purpose, and importance.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tone": map[string]any{
							"type":        "string",
							"description": "One word like: neutral, polite, rude, sarcastic",
						},
						"verbosity": map[string]any{
							"type":        "string",
							"description": "One word like: short, normal, detailed",
						},
						"formality": map[string]any{
							"type":        "string",
							"description": "One word like: casual, neutral, formal",
						},
						"language": map[string]any{
							"type":        "string",
							"description": "One word like: auto, english, russian",
						},
						"emotion": map[string]any{
							"type":        "string",
							"description": "One word like: neutral, happy, angry, sarcastic",
						},
						"humorLevel": map[string]any{
							"type":        "string",
							"description": "One word like: none, low, medium, high",
						},
						"detailLevel": map[string]any{
							"type":        "string",
							"description": "One word like: short, normal, detailed",
						},
						"knowledgeFocus": map[string]any{
							"type":        "string",
							"description": "One word like: technical, creative, general, personal",
						},
						"confidenceMode": map[string]any{
							"type":        "string",
							"description": "One word like: safe, assertive, speculative",
						},
						"politenessLevel": map[string]any{
							"type":        "string",
							"description": "One word like: low, medium, high",
						},
						"personalityProfile": map[string]any{
							"type":        "string",
							"description": "One word like: techer, developer, actor",
						},
					},
					"required": []string{},
				},
			},
		},
	}
}

func GetToolDefault() []models.Tool {
	return []models.Tool{
		{
			Type: "function",
			Function: models.Function{
				Name:        "change_agent_settings",
				Description: "Fucktion for change agent persoanalization.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tone": map[string]any{
							"type":        "string",
							"description": "One word like: neutral, polite, rude, sarcastic (custom allowed)",
						},
						"verbosity": map[string]any{
							"type":        "string",
							"description": "One word like: short, normal, detailed (custom allowed)",
						},
						"formality": map[string]any{
							"type":        "string",
							"description": "One word like: casual, neutral, formal (custom allowed)",
						},
						"language": map[string]any{
							"type":        "string",
							"description": "One word like: auto, english, russian (custom allowed)",
						},
						"emotion": map[string]any{
							"type":        "string",
							"description": "One word like: neutral, happy, angry, sarcastic (custom allowed)",
						},
						"humorLevel": map[string]any{
							"type":        "string",
							"description": "One word like: none, low, medium, high (custom allowed)",
						},
						"detailLevel": map[string]any{
							"type":        "string",
							"description": "One word like: short, normal, detailed (custom allowed)",
						},
						"knowledgeFocus": map[string]any{
							"type":        "string",
							"description": "One word like: technical, creative, general, personal (custom allowed)",
						},
						"confidenceMode": map[string]any{
							"type":        "string",
							"description": "One word like: safe, assertive, speculative (custom allowed)",
						},
						"politenessLevel": map[string]any{
							"type":        "string",
							"description": "One word like: low, medium, high (custom allowed)",
						},
						"personalityProfile": map[string]any{
							"type":        "string",
							"description": "One word like: techer, developer, actor (custom allowed)",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Function: models.Function{
				Name:        "agent_mode",
				Description: "Enter reasoning agent mode and perform an action",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"thought": map[string]any{
							"type":        "string",
							"description": "Internal reasoning step explaining what the agent plans to do next",
						},

						"action": map[string]any{
							"type": "object",
							"properties": map[string]any{

								"function": map[string]any{
									"type":        "string",
									"description": "Name of the tool to call",
								},

								"args": map[string]any{
									"type":                 "object",
									"description":          "Arguments for the tool",
									"additionalProperties": true,
								},
							},
						},
					},
					"required": []string{"thought"},
				},
			},
		},
	}
}

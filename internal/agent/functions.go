package agent

import "github.com/nikaydo/personal-assistant/internal/models"

func GetAgentTool() []models.Tool {
	return []models.Tool{
		{
			Type: "function",
			Function: models.Function{
				Name:        "reasoning",
				Description: "Reasoning agent mode and perform an action",
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
								"arguments": map[string]any{
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
		{
			Type: "function",
			Function: models.Function{
				Name:        "stop",
				Description: "Finish the reasoning process and return the final answer to the user.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"response": map[string]any{
							"type":        "string",
							"description": "Final answer for the user",
						},
					},
					"required": []string{"response"},
				},
			},
		},
	}
}

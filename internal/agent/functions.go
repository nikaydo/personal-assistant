package agent

import "github.com/nikaydo/personal-assistant/internal/models"

func GetAgentTool() []models.Tool {
	return []models.Tool{
		{
			Type: "function",
			Function: models.Function{
				Name:        "reasoning",
				Description: "Reason about the next step and optionally perform an action",
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
									"description": "Name of the tool to call. Use literal 'command' (without namespace prefixes).",
								},
								"mode": map[string]any{
									"type":        "string",
									"enum":        []string{"exec", "shell"},
									"description": "Execution mode. Use exec by default. Use shell only when shell operators/heredoc/redirection are required.",
								},
								"arguments": map[string]any{
									"oneOf": []map[string]any{
										{
											"type":                 "object",
											"additionalProperties": true,
										},
										{
											"type": "string",
										},
										{
											"type":  "array",
											"items": map[string]any{"type": "string"},
										},
									},
									"description": "Primary action payload. Supported forms: object {command,args,mode}, command-line string, or string array [command,arg1,...]. For multiline text/file writes prefer mode=shell with printf or heredoc (cat <<'EOF').",
								},
								"args": map[string]any{
									"oneOf": []map[string]any{
										{
											"type":  "array",
											"items": map[string]any{"type": "string"},
										},
										{
											"type":                 "object",
											"additionalProperties": true,
										},
									},
									"description": "Backward-compatible alias for arguments. Do not send both arguments and args in one action.",
								},
							},
							"required": []string{"function"},
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
		{
			Type: "function",
			Function: models.Function{
				Name:        "command",
				Description: "Execute a shell command on the host",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"command": map[string]any{
							"type":        "string",
							"description": "Command name like pwd, cd, ls with args",
						},
						"args": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "List of arguments",
						},
						"mode": map[string]any{
							"type":        "string",
							"enum":        []string{"exec", "shell"},
							"description": "Execution mode. Default is exec. Shell mode runs 'sh -c <command>' explicitly.",
						},
					},
					"required": []string{"command"},
				},
			},
		},
	}
}

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
				Description: "Update agent personalization settings.",
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
				Description: "Update agent personalization settings.",
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
				Description: "Enter reasoning agent mode for any request that needs tools or actions (files, commands, external actions). Provide the original user request in 'question'.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question": map[string]any{
							"type":        "string",
							"description": "Original user request to solve",
						},
						"thought": map[string]any{
							"type":        "string",
							"description": "Internal reasoning step explaining what the agent plans to do next",
						},
					},
					"required": []string{"thought"},
				},
			},
		},
	}
}

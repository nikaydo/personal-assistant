package ai

// func GetToolRouter() []models.Tool {
// 	return []models.Tool{
// 		{
// 			Type: "function",
// 			Function: models.Function{
// 				Name:        "select_tool_group",
// 				Description: "Select which tool group should be used",
// 				Parameters: map[string]any{
// 					"type": "object",
// 					"properties": map[string]any{
// 						"group": map[string]any{
// 							"type": "string",
// 							"enum": []string{
// 								"jira",
// 								"spotify",
// 							},
// 							"description": "Tool group. Use 'jira' for creating, searching, and deleting Jira projects. Use 'spotify' for play music on Spotify.",
// 						},
// 					},
// 					"required": []string{"group"},
// 				},
// 			},
// 		},
// 	}
// }

package ai

import "github.com/nikaydo/jira-filler/internal/models"

type ToolConf struct {
	AccountId string `json:"account_id,omitempty"`
}

func GetToolRouter() []models.Tool {
	return []models.Tool{
		{
			Type: "function",
			Function: models.Function{
				Name:        "select_tool_group",
				Description: "Select which tool group should be used",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"group": map[string]any{
							"type": "string",
							"enum": []string{
								"jira",
							},
							"description_map": map[string]string{
								"jira": "Tool for tracking tasks in Jira. include only functions for creating project, searching projects and deleting projects.",
							},
						},
					},
					"required": []string{"group"},
				},
			},
		},
	}
}

func GetToolsJira(conf ToolConf) []models.Tool {
	return []models.Tool{
		{
			Type: "function",
			Function: models.Function{
				Name:        "create_project",
				Description: "Create project in Jira",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"notificationScheme": map[string]any{
							"type":        "string",
							"description": "The ID of the notification scheme for the project.",
						},
						"fieldConfigurationScheme": map[string]any{
							"type":        "string",
							"description": "The ID of the field configuration scheme for the project.",
						},
						"issueSecurityScheme": map[string]any{
							"type":        "string",
							"description": "The ID of the issue security scheme for the project.",
						},
						"permissionScheme": map[string]any{
							"type":        "string",
							"description": "The ID of the permission scheme for the project.",
						},
						"issueTypeScheme": map[string]any{
							"type":        "string",
							"description": "The ID of the issue type scheme for the project.",
						},
						"issueTypeScreenScheme": map[string]any{
							"type":        "string",
							"description": "The ID of the issue type screen scheme for the project.",
						},
						"workflowScheme": map[string]any{
							"type":        "string",
							"description": "The ID of the workflow scheme for the project.",
						},
						"description": map[string]any{
							"type":        "string",
							"description": "The description of the project.",
						},
						"leadAccountId": map[string]any{
							"type":        "string",
							"enum":        []string{conf.AccountId},
							"description": "The account ID of the lead for the project.",
						},
						"url": map[string]any{
							"type":        "string",
							"description": "The URL of the project.",
						},
						"projectTemplateKey": map[string]any{
							"type": "string",
							"enum": []string{
								"com.atlassian.jira-core-project-templates:jira-core-simplified-content-management",
								"com.atlassian.jira-core-project-templates:jira-core-simplified-document-approval",
								"com.atlassian.jira-core-project-templates:jira-core-simplified-lead-tracking",
								"com.atlassian.jira-core-project-templates:jira-core-simplified-process-control",
								"com.atlassian.jira-core-project-templates:jira-core-simplified-procurement",
								"com.atlassian.jira-core-project-templates:jira-core-simplified-project-management",
								"com.atlassian.jira-core-project-templates:jira-core-simplified-recruitment",
								"com.atlassian.jira-core-project-templates:jira-core-simplified-task-tracking",
								"com.atlassian.servicedesk:simplified-it-service-desk",
								"com.atlassian.servicedesk:simplified-internal-service-desk",
								"com.atlassian.servicedesk:simplified-external-service-desk",
								"com.pyxis.greenhopper.jira:gh-simplified-agility-kanban",
								"com.pyxis.greenhopper.jira:gh-simplified-agility-scrum",
								"com.pyxis.greenhopper.jira:gh-simplified-kanban-classic",
								"com.pyxis.greenhopper.jira:gh-simplified-scrum-classic",
							},
							"description": "The project template key.",
						},
						"avatarId": map[string]any{
							"type":        "string",
							"description": "The avatar ID of the project.",
						},
						"name": map[string]any{
							"type":        "string",
							"description": "The name of the project.",
						},
						"assigneeType": map[string]any{
							"type":        "string",
							"description": "The assignee type of the project.",
						},
						"projectTypeKey": map[string]any{
							"type":        "string",
							"description": "The project type key.",
						},
						"key": map[string]any{
							"type":        "string",
							"description": "The key of the project.",
						},
						"categoryId": map[string]any{
							"type":        "int",
							"description": "The category ID of the project.",
						},
					},
					"required": []string{"key", "name", "leadAccountId", "projectTemplateKey"},
				},
			},
		},
		{
			Type: "function",
			Function: models.Function{
				Name:        "search_projects",
				Description: "Search returns a paginated list of projects visible to the user.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"options": map[string]any{
							"type":        "object",
							"description": "query parameters",
							"items": map[string]any{
								"orderBy": map[string]any{
									"type":        "string",
									"description": "The order by field for the search.",
								},
								"ids": map[string]any{
									"type":        "array",
									"description": "The project IDs to filter the results by. To include multiple IDs, provide an ampersand-separated list.",
									"items": map[string]any{
										"type": "integer",
									},
								},
								"keys": map[string]any{
									"type":        "array",
									"description": "The project keys to filter the results by. To include multiple keys, provide an ampersand-separated list.",
									"items": map[string]any{
										"type": "string",
									},
								},
								"query": map[string]any{
									"type":        "string",
									"description": " Filter the results using a literal string. Projects with a matching key or name are returned (case-insensitive).",
								},
								"typekeys": map[string]any{
									"type":        "array",
									"description": "Orders results by the project type. This parameter accepts a comma-separated list.  Valid values are business, service_desk, and software.",
									"items": map[string]any{
										"type": "string",
									},
								},
								"categoryid": map[string]any{
									"type":        "int",
									"description": "The ID of the project's category. A complete list of category IDs is found using the Get all project categories operation.",
								},
								"action": map[string]any{
									"type":        "string",
									"description": "The action to perform on the search results.",
								},
							},
						},
						"startAt": map[string]any{
							"type":        "int",
							"description": "The index of the first project to return in the results.",
						},
						"maxResults": map[string]any{
							"type":        "int",
							"description": "The maximum number of projects to return in the results.",
						},
					},
					"required": []string{"startAt", "maxResults"},
				},
			},
		},
		{
			Type: "function",
			Function: models.Function{
				Name:        "delete_project",
				Description: "Delete a project. You can't delete a project if it's archived. To delete an archived project, restore the project and then delete it. To restore a project, use the Jira UI.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"projectKeyOrID": map[string]any{
							"type":        "string",
							"description": "The project key or ID to delete.",
						},
					},
					"required": []string{"projectKeyOrID"},
				},
			},
		},
	}
}

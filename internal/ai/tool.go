package ai

import "github.com/nikaydo/personal-assistant/internal/models"

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
							"description": "Tool group. Use 'jira' for creating, searching, and deleting Jira projects.",
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
							"description": "The key of the project in upper case without any symbols like '-' ',' '.' .",
						},
						"categoryId": map[string]any{
							"type":        "integer",
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
									"type":        "integer",
									"description": "The ID of the project's category. A complete list of category IDs is found using the Get all project categories operation.",
								},
								"action": map[string]any{
									"type":        "string",
									"description": "The action to perform on the search results.",
								},
							},
						},
						"startAt": map[string]any{
							"type":        "integer",
							"description": "The index of the first project to return in the results.",
						},
						"maxResults": map[string]any{
							"type":        "integer",
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
		{
			Type: "function",
			Function: models.Function{
				Name:        "issueOne",
				Description: "Build a complete Jira issue payload for models.IssueSchemeV2. Always fill fields.summary, fields.project, and fields.issuetype. Put Jira custom fields into customFields using keys like customfield_10010.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"issue": map[string]any{
							"type":        "object",
							"description": "Payload mapped to models.IssueSchemeV2.",
							"properties": map[string]any{
								"fields": map[string]any{
									"type":        "object",
									"description": "Issue fields from models.IssueFieldsSchemeV2.",
									"properties": map[string]any{
										"summary": map[string]any{
											"type":        "string",
											"description": "Short issue title.",
										},
										"description": map[string]any{
											"type":        "string",
											"description": "Detailed issue description.",
										},
										"project": map[string]any{
											"type":        "object",
											"description": "Target Jira project. Prefer key, optionally id.",
											"properties": map[string]any{
												"key": map[string]any{
													"type":        "string",
													"description": "Project key, for example ENG.",
												},
												"id": map[string]any{
													"type":        "string",
													"description": "Project ID as string.",
												},
											},
											"required": []string{"key"},
										},
										"issuetype": map[string]any{
											"type":        "object",
											"description": "Jira issue type. Prefer id, optionally name.",
											"properties": map[string]any{
												"id": map[string]any{
													"type":        "string",
													"description": "Issue type ID.",
												},
												"name": map[string]any{
													"type":        "string",
													"description": "Issue type name, for example Task or Bug.",
												},
											},
										},
										"priority": map[string]any{
											"type":        "object",
											"description": "Issue priority.",
											"properties": map[string]any{
												"id": map[string]any{
													"type": "string",
												},
												"name": map[string]any{
													"type": "string",
												},
											},
										},
										"assignee": map[string]any{
											"type":        "object",
											"description": "Assignee user.",
											"properties": map[string]any{
												"accountId": map[string]any{
													"type": "string",
												},
											},
										},
										"reporter": map[string]any{
											"type":        "object",
											"description": "Reporter user.",
											"properties": map[string]any{
												"accountId": map[string]any{
													"type": "string",
												},
											},
										},
										"labels": map[string]any{
											"type": "array",
											"items": map[string]any{
												"type": "string",
											},
										},
										"duedate": map[string]any{
											"type":        "string",
											"description": "Due date in format YYYY-MM-DD.",
										},
										"parent": map[string]any{
											"type":        "object",
											"description": "Parent issue for subtasks.",
											"properties": map[string]any{
												"key": map[string]any{
													"type": "string",
												},
												"id": map[string]any{
													"type": "string",
												},
											},
										},
										"components": map[string]any{
											"type": "array",
											"items": map[string]any{
												"type": "object",
												"properties": map[string]any{
													"id": map[string]any{
														"type": "string",
													},
													"name": map[string]any{
														"type": "string",
													},
												},
											},
										},
										"versions": map[string]any{
											"type": "array",
											"items": map[string]any{
												"type": "object",
												"properties": map[string]any{
													"id": map[string]any{
														"type": "string",
													},
													"name": map[string]any{
														"type": "string",
													},
												},
											},
										},
										"fixVersions": map[string]any{
											"type": "array",
											"items": map[string]any{
												"type": "object",
												"properties": map[string]any{
													"id": map[string]any{
														"type": "string",
													},
													"name": map[string]any{
														"type": "string",
													},
												},
											},
										},
									},
									"required": []string{"summary", "project", "issuetype"},
								},
							},
							"required": []string{"fields"},
						},
						"customFields": map[string]any{
							"type":        "object",
							"description": "Custom Jira fields map. Key format: customfield_<id>. Value can be string, number, boolean, object, array, or null depending on Jira field type.",
							"patternProperties": map[string]any{
								"^customfield_[0-9]+$": map[string]any{
									"type": []string{"string", "number", "boolean", "object", "array", "null"},
								},
							},
							"additionalProperties": true,
						},
					},
					"required": []string{"issue"},
				},
			},
		},
	}
}

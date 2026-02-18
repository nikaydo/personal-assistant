package jira

import "github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"

type Issue struct {
	Summary      string                  `json:"summary,omitempty"`
	Project      string                  `json:"project,omitempty"`
	IssueType    *models.IssueTypeScheme `json:"issueType,omitempty"`
	CustomFields map[string]any          `json:"customFields,omitempty"`
}

func (i *Issue) GetCustomFields() string {

}

func (j *Jira) CreateIssueOne() {
	var payload = models.IssueSchemeV2{
		Fields: &models.IssueFieldsSchemeV2{},
	}
}

func GetToolSumm() []models.Tool {
	return []models.Tool{
		{
			Type: "function",
			Function: models.Function{
				Name:        "summarize",
				Description: "A function for creating a short summary of a text based on its category, purpose, and importance.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"summary": map[string]any{
							"type":        "string",
							"description": "Summarized text according to the history of previous queries",
						},
						"project": map[string]any{
							"type":        "string",
							"description": "Summarized text according to the history of previous queries",
						},
						"issueType": map[string]any{
							"type":        "object",
							"description": "Summarized text according to the history of previous queries",
							"items": map[string]any{
								"type": "string",
							},
						},
						"customFields": map[string]any{
							"type":        "object",
							"description": "Summarized text according to the history of previous queries",
							"items":       map[string]any{},
						},
					},
					"required": []string{"summary", "project", "issueType"},
				},
			},
		},
	}
}

func (j *Jira) CreateIssue() {

}

func (j *Jira) GetIssue() {

}

func (j *Jira) DeleteIssue() {

}

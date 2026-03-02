package jira

import (
	"context"
	"encoding/json"
	"fmt"

	atlassianmodels "github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

type Issue struct {
	Summary      string                           `json:"summary,omitempty"`
	Project      string                           `json:"project,omitempty"`
	IssueType    *atlassianmodels.IssueTypeScheme `json:"issueType,omitempty"`
	CustomFields map[string]any                   `json:"customFields,omitempty"`
}

func (i *Issue) GetCustomFields() string {
	if len(i.CustomFields) == 0 {
		return "{}"
	}
	b, err := json.Marshal(i.CustomFields)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func customFieldsFromMap(fields map[string]any) (*atlassianmodels.CustomFields, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	cf := &atlassianmodels.CustomFields{}
	for key, value := range fields {
		if err := cf.Raw(key, value); err != nil {
			return nil, fmt.Errorf("jira custom field %q: %w", key, err)
		}
	}

	return cf, nil
}

func textToADFNode(text string) *atlassianmodels.CommentNodeScheme {
	if text == "" {
		return nil
	}

	return &atlassianmodels.CommentNodeScheme{
		Version: 1,
		Type:    "doc",
		Content: []*atlassianmodels.CommentNodeScheme{
			{
				Type: "paragraph",
				Content: []*atlassianmodels.CommentNodeScheme{
					{
						Type: "text",
						Text: text,
					},
				},
			},
		},
	}
}

func toIssueSchemeV3(payload *atlassianmodels.IssueSchemeV2) *atlassianmodels.IssueScheme {
	if payload == nil {
		return nil
	}

	var fields *atlassianmodels.IssueFieldsScheme
	if payload.Fields != nil {
		fields = &atlassianmodels.IssueFieldsScheme{
			Parent:      payload.Fields.Parent,
			IssueType:   payload.Fields.IssueType,
			IssueLinks:  payload.Fields.IssueLinks,
			Versions:    payload.Fields.Versions,
			Project:     payload.Fields.Project,
			FixVersions: payload.Fields.FixVersions,
			Priority:    payload.Fields.Priority,
			Components:  payload.Fields.Components,
			Reporter:    payload.Fields.Reporter,
			Assignee:    payload.Fields.Assignee,
			Summary:     payload.Fields.Summary,
			Labels:      payload.Fields.Labels,
			Security:    payload.Fields.Security,
			DueDate:     payload.Fields.DueDate,
			Description: textToADFNode(payload.Fields.Description),
		}
	}

	return &atlassianmodels.IssueScheme{
		ID:             payload.ID,
		Key:            payload.Key,
		Self:           payload.Self,
		Transitions:    payload.Transitions,
		Changelog:      payload.Changelog,
		Fields:         fields,
		RenderedFields: payload.RenderedFields,
	}
}

func (j *Jira) CreateIssueOne(payload *atlassianmodels.IssueSchemeV2, customFields map[string]any) (*atlassianmodels.IssueResponseScheme, *atlassianmodels.ResponseScheme, error) {
	issuePayload := toIssueSchemeV3(payload)
	if issuePayload == nil || issuePayload.Fields == nil {
		return nil, nil, fmt.Errorf("jira: issue payload and issue fields are required")
	}

	fields, err := customFieldsFromMap(customFields)
	if err != nil {
		return nil, nil, err
	}

	return j.Client.Issue.Create(context.Background(), issuePayload, fields)
}
func (j *Jira) CreateIssue() {

}

func (j *Jira) GetIssue() {

}

func (j *Jira) DeleteIssue() {

}

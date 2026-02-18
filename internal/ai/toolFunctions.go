package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	mod "github.com/nikaydo/personal-assistant/internal/models"
)

func (ai *Ai) createProjectJira(resp mod.ResponseBody) (string, error) {
	tc, err := firstToolCall(resp)
	if err != nil {
		return "", err
	}
	var pj *models.ProjectPayloadScheme
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &pj); err != nil {
		return "", err
	}
	ai.Logger.Info(
		"createProjectJira: executing",
		"key", pj.Key,
		"name", pj.Name,
		"template", pj.ProjectTemplateKey,
	)
	projectScheme, responseScheme, err := ai.Jira.CreateProject(pj)
	if err != nil {
		if isProjectKeyConflict(responseScheme) {
			oldKey := pj.Key
			pj.Key = nextProjectKey(oldKey)
			ai.Logger.Warn("createProjectJira: project key conflict, retrying", "oldKey", oldKey, "newKey", pj.Key)
			projectScheme, responseScheme, err = ai.Jira.CreateProject(pj)
			if err == nil {
				return fmt.Sprintf("function return: %+v. with result data: %+v (key changed from %s to %s due to conflict)", projectScheme, responseScheme, oldKey, pj.Key), nil
			}
		}

		ai.Logger.Error(
			"createProjectJira: jira create failed:",
			err,
			"status", responseStatusCode(responseScheme),
			"jira_error", jiraErrorMessage(responseScheme),
			"response", fmt.Sprintf("%+v", responseScheme),
		)
		return "", err
	}

	return fmt.Sprintf("function return: %+v. with result data: %+v", projectScheme, responseScheme), nil
}

func responseStatusCode(resp *models.ResponseScheme) int {
	if resp == nil {
		return 0
	}
	return resp.Code
}

type jiraErrorResponse struct {
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
}

func jiraErrorMessage(resp *models.ResponseScheme) string {
	if resp == nil {
		return ""
	}
	raw := strings.TrimSpace(resp.Bytes.String())
	if raw == "" {
		return ""
	}

	var parsed jiraErrorResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return raw
	}

	parts := make([]string, 0, len(parsed.ErrorMessages)+len(parsed.Errors))
	parts = append(parts, parsed.ErrorMessages...)
	for field, msg := range parsed.Errors {
		parts = append(parts, field+": "+msg)
	}
	return strings.Join(parts, "; ")
}

func isProjectKeyConflict(resp *models.ResponseScheme) bool {
	if resp == nil {
		return false
	}
	raw := strings.TrimSpace(resp.Bytes.String())
	if raw == "" {
		return false
	}

	var parsed jiraErrorResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return false
	}

	msg, ok := parsed.Errors["projectKey"]
	if !ok {
		return false
	}

	lower := strings.ToLower(msg)
	return strings.Contains(lower, "already") || strings.Contains(lower, "уже")
}

func nextProjectKey(base string) string {
	clean := strings.Builder{}
	for _, r := range strings.ToUpper(base) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			clean.WriteRune(r)
		}
	}
	prefix := clean.String()
	if prefix == "" {
		prefix = "PRJ"
	}
	if len(prefix) > 7 {
		prefix = prefix[:7]
	}
	suffix := time.Now().Unix() % 1000
	return fmt.Sprintf("%s%03d", prefix, suffix)
}

func (ai *Ai) searchProjectJira(resp mod.ResponseBody) (string, error) {
	tc, err := firstToolCall(resp)
	if err != nil {
		return "", err
	}
	var pj mod.ProjectSearchOptions
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &pj); err != nil {
		return "", err
	}
	ai.Logger.Info(
		"searchProjectJira: executing",
		"query", pj.Options.Query,
		"startAt", pj.StartAt,
		"maxResults", pj.MaxResults,
	)
	opt := &models.ProjectSearchOptionsScheme{
		Query:      pj.Options.Query,
		OrderBy:    pj.Options.OrderBy,
		Keys:       pj.Options.Keys,
		IDs:        pj.Options.IDs,
		TypeKeys:   pj.Options.TypeKeys,
		CategoryID: pj.Options.CategoryID,
		Action:     pj.Options.Action,
	}
	projectScheme, responseScheme, err := ai.Jira.SearchProject(opt, pj.StartAt, pj.MaxResults)
	if err != nil {
		ai.Logger.Error("searchProjectJira: jira search failed:", err)
		return "", err
	}
	ai.Logger.Info("searchProjectJira: success", "returned_projects", len(projectScheme.Values))
	var val string
	for _, v := range projectScheme.Values {
		val += fmt.Sprintf("%+v", v)
	}

	return fmt.Sprintf("function return: %+v. with result data: %b", val, responseScheme.Body), nil
}

func (ai *Ai) deleteProjectJira(resp mod.ResponseBody) (string, error) {
	tc, err := firstToolCall(resp)
	if err != nil {
		return "", err
	}
	var pj struct {
		ProjectKeyOrID string `json:"projectKeyOrID"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &pj); err != nil {
		return "", err
	}
	ai.Logger.Info("deleteProjectJira: executing", "project_key_or_id", pj.ProjectKeyOrID)
	responseScheme, err := ai.Jira.DeleteProject(pj.ProjectKeyOrID)
	if err != nil {
		ai.Logger.Error("deleteProjectJira: jira delete failed:", err)
		return "", err
	}
	ai.Logger.Info("deleteProjectJira: success", "status", responseStatusCode(responseScheme))

	return fmt.Sprintf("function return: %+v", responseScheme), nil
}

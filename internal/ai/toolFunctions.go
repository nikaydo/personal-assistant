package ai

import (
	"encoding/json"
	"fmt"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	mod "github.com/nikaydo/jira-filler/internal/models"
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
	projectScheme, responseScheme, err := ai.Jira.CreateProject(pj)
	if err != nil {
		ai.Logger.Error(err.Error())
		return "", err
	}

	return fmt.Sprintf("function return: %+v. with result data: %+v", projectScheme, responseScheme), nil
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
		ai.Logger.Error(err.Error())
		return "", err
	}
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
	responseScheme, err := ai.Jira.DeleteProject(pj.ProjectKeyOrID)
	if err != nil {
		ai.Logger.Error(err.Error())
		return "", err
	}

	return fmt.Sprintf("function return: %+v", responseScheme), nil
}

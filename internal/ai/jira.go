package ai

import (
	"fmt"
	"slices"

	mod "github.com/nikaydo/jira-filler/internal/models"
)

var PojectFunc []string = []string{
	"create_project",
	"search_projects",
	"delete_project",
}

func (ai *Ai) JiraTasks(q mod.Message) (mod.ResponseBody, error) {
	msg := ai.makeMessage(ai.Config.PromtSystemChat)
	msg = append(msg, q)

	resp, err := ai.Ask(msg, GetToolsJira(*ai.ToolConf))
	if err != nil {
		return mod.ResponseBody{}, err
	}
	ai.Logger.Task("JiraTasks", resp)
	if len(resp.Choices[0].Message.ToolCalls) > 0 {
		//check if fucnction exist in project
		if slices.Contains(PojectFunc, resp.Choices[0].Message.ToolCalls[0].Function.Name) {
			r, err := ai.JiraTasksProject(q, resp)
			if err != nil {
				return mod.ResponseBody{}, err
			}
			return r, nil
		}
	}
	return resp, nil
}

func (ai *Ai) JiraTasksProject(q mod.Message, resp mod.ResponseBody) (mod.ResponseBody, error) {
	switch resp.Choices[0].Message.ToolCalls[0].Function.Name {
	case "create_project":
		r, err := ai.createProjectJira(resp)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		resp, err := ai.backAsk(resp.Choices[0].Message.ToolCalls[0].ID, r, q)
		if err != nil {
			return mod.ResponseBody{}, err
		}

		ai.Logger.Task("backAsk", resp)
		return resp, nil
	case "search_projects":
		r, err := ai.searchProjectJira(resp)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		resp, err := ai.backAsk(resp.Choices[0].Message.ToolCalls[0].ID, r, q)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		ai.Logger.Task("backAsk", resp)
		return resp, nil
	case "delete_project":
		r, err := ai.deleteProjectJira(resp)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		resp, err := ai.backAsk(resp.Choices[0].Message.ToolCalls[0].ID, r, q)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		ai.Logger.Task("backAsk", resp)
		return resp, nil
	}
	return mod.ResponseBody{}, fmt.Errorf("unknown tool function in jira: %v", resp.Choices[0].Message.ToolCalls[0].Function.Name)
}

func (ai *Ai) backAsk(id, content string, q mod.Message) (mod.ResponseBody, error) {
	msg := mod.Message{
		Role:       "tool",
		ToolCallID: id,
		Content:    content,
	}
	m := ai.makeMessage(ai.Config.PromtSystemChat)
	m = append(m, msg)

	resp, err := ai.Ask(m, GetToolRouter())
	if err != nil {
		return mod.ResponseBody{}, err
	}
	ai.Memory.History = append(ai.Memory.History, Question{Q: q.Content, A: resp.Choices[0].Message.Content, ContextToken: resp.Usage.TotalTokens})
	return resp, nil
}

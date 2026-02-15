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
		ai.Logger.Error(fmt.Sprintf("jira tasks ask failed: %v", err))
		return mod.ResponseBody{}, err
	}
	ai.Logger.Task("JiraTasks", resp)
	tc, err := firstToolCall(resp)
	if err != nil {
		msgChoice, msgErr := firstChoice(resp)
		if msgErr == nil && msgChoice.Content != "" {
			return resp, nil
		}
		ai.Logger.Error(err.Error())
		return mod.ResponseBody{}, err
	}

	//check if function exist in project
	if slices.Contains(PojectFunc, tc.Function.Name) {
		r, err := ai.JiraTasksProject(q, resp)
		if err != nil {
			ai.Logger.Error(fmt.Sprintf("jira project tool failed: %v", err))
			return mod.ResponseBody{}, err
		}
		return r, nil
	}

	err = fmt.Errorf("unsupported jira tool function: %s", tc.Function.Name)
	ai.Logger.Error(err.Error())
	return mod.ResponseBody{}, err
}

func (ai *Ai) JiraTasksProject(q mod.Message, resp mod.ResponseBody) (mod.ResponseBody, error) {
	tc, err := firstToolCall(resp)
	if err != nil {
		ai.Logger.Error(err.Error())
		return mod.ResponseBody{}, err
	}

	switch tc.Function.Name {
	case "create_project":
		r, err := ai.createProjectJira(resp)
		if err != nil {
			ai.Logger.Error(fmt.Sprintf("create project jira failed: %v -||||- %+v", err, resp))
			return ai.ResponseFailed(fmt.Sprintf("You execute tool create_project in jira and its failed! Param: %+v | Result: %+v", tc.Function.Arguments, r), resp)
		}
		resp, err := ai.backAsk(tc.ID, r, q)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		ai.Logger.Task("backAsk", resp)
		return resp, nil
	case "search_projects":
		r, err := ai.searchProjectJira(resp)
		if err != nil {
			ai.Logger.Error(fmt.Sprintf("search project jira failed: %v -||||- %+v", err, resp))
			return ai.ResponseFailed(fmt.Sprintf("You execute tool search_projects in jira and its failed! Param: %+v | Result: %+v", tc.Function.Arguments, r), resp)
		}
		resp, err := ai.backAsk(tc.ID, r, q)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		ai.Logger.Task("backAsk", resp)
		return resp, nil
	case "delete_project":
		r, err := ai.deleteProjectJira(resp)
		if err != nil {
			ai.Logger.Error(fmt.Sprintf("delete project jira failed: %v -||||- %+v", err, resp))
			return ai.ResponseFailed(fmt.Sprintf("You execute tool delete_project in jira and its failed! Param: %+v | Result: %+v", tc.Function.Arguments, r), resp)
		}
		resp, err := ai.backAsk(tc.ID, r, q)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		ai.Logger.Task("backAsk", resp)
		return resp, nil
	}

	return ai.ResponseFailed(fmt.Sprintf("You execute unknown tool function in jira: %s", tc.Function.Name), resp)
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
		ai.Logger.Error(fmt.Sprintf("backAsk failed: %v", err))
		return mod.ResponseBody{}, err
	}
	msgChoice, err := firstChoice(resp)
	if err != nil {
		ai.Logger.Error(err.Error())
		return mod.ResponseBody{}, err
	}
	ai.Memory.History = append(ai.Memory.History, Question{Q: q.Content, A: msgChoice.Content, ContextToken: resp.Usage.TotalTokens})
	return resp, nil
}

func (ai *Ai) ResponseFailed(content string, resp mod.ResponseBody) (mod.ResponseBody, error) {
	failedstring := "Failed execute function: " + content
	msg := mod.Message{
		Role:    "user",
		Content: failedstring,
	}
	m := ai.makeMessage(ai.Config.PromtSystemChat)
	m = append(m, msg)
	resp, err := ai.Ask(m, GetToolRouter())
	if err != nil {
		ai.Logger.Error(fmt.Sprintf("backAsk failed: %v", err))
		return mod.ResponseBody{}, err
	}
	msgChoice, err := firstChoice(resp)
	if err != nil {
		ai.Logger.Error(err.Error())
		return mod.ResponseBody{}, err
	}
	ai.Memory.History = append(ai.Memory.History, Question{Q: failedstring, A: msgChoice.Content, ContextToken: resp.Usage.TotalTokens})
	return resp, nil
}

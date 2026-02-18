package ai

import (
	"fmt"
	"slices"

	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	mod "github.com/nikaydo/personal-assistant/internal/models"
)

var PojectFunc []string = []string{
	"create_project",
	"search_projects",
	"delete_project",
}

func (ai *Ai) JiraTasks(q mod.Message) (mod.ResponseBody, error) {
	resp, err := llmcalls.Ask(ai.makeBody(ai.Memory.HistoryMessage(q, ai.Config.PromtSystemChat), GetToolsJira(*ai.ToolConf)), ai.Config)
	if err != nil {
		ai.Logger.Error("JiraTasks: ask failed:", err)
		return mod.ResponseBody{}, err
	}
	ai.Logger.Task("JiraTasks", resp)
	tc, err := firstToolCall(resp)
	if err != nil {
		msgChoice, msgErr := firstChoice(resp)
		if msgErr == nil && msgChoice.Content != "" {
			return resp, nil
		}
		ai.Logger.Error("JiraTasks: firstToolCall failed:", err)
		return mod.ResponseBody{}, err
	}

	if slices.Contains(PojectFunc, tc.Function.Name) {
		r, err := ai.JiraTasksProject(q, resp)
		if err != nil {
			ai.Logger.Error("JiraTasks: jira project tool failed:", err)
			return mod.ResponseBody{}, err
		}
		return r, nil
	}

	err = fmt.Errorf("unsupported jira tool function: %s", tc.Function.Name)
	ai.Logger.Error("JiraTasks:", err)
	return mod.ResponseBody{}, err
}

func (ai *Ai) JiraTasksProject(q mod.Message, resp mod.ResponseBody) (mod.ResponseBody, error) {
	tc, err := firstToolCall(resp)
	if err != nil {
		ai.Logger.Error("JiraTasksProject: firstToolCall failed:", err)
		return mod.ResponseBody{}, err
	}
	msgChoice, err := firstChoice(resp)
	if err != nil {
		ai.Logger.Error("JiraTasksProject: firstChoice failed:", err)
		return mod.ResponseBody{}, err
	}

	switch tc.Function.Name {
	case "create_project":
		r, err := ai.createProjectJira(resp)
		if err != nil {
			ai.Logger.Error("JiraTasksProject: create project jira failed:", err, "resp:", resp)
			return ai.ResponseFailed(fmt.Sprintf("You execute tool create_project in jira and its failed! Param: %+v | Result: %+v", tc.Function.Arguments, r), resp)
		}
		ai.Logger.Info("Respose from create_project", r)
		resp, err := ai.backAsk(tc.ID, r, q, msgChoice)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		ai.Logger.Task("backAsk", resp)
		return resp, nil
	case "search_projects":
		r, err := ai.searchProjectJira(resp)
		if err != nil {
			ai.Logger.Error("JiraTasksProject: search project jira failed:", err, "resp:", resp)
			return ai.ResponseFailed(fmt.Sprintf("You execute tool search_projects in jira and its failed! Param: %+v | Result: %+v", tc.Function.Arguments, r), resp)
		}
		ai.Logger.Info("Respose from search_projects", r)
		resp, err := ai.backAsk(tc.ID, r, q, msgChoice)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		ai.Logger.Task("backAsk", resp)
		return resp, nil
	case "delete_project":
		r, err := ai.deleteProjectJira(resp)
		if err != nil {
			ai.Logger.Error("JiraTasksProject: delete project jira failed:", err, "resp:", resp)
			return ai.ResponseFailed(fmt.Sprintf("You execute tool delete_project in jira and its failed! Param: %+v | Result: %+v", tc.Function.Arguments, r), resp)
		}
		ai.Logger.Info("Respose from delete_project", r)
		resp, err := ai.backAsk(tc.ID, r, q, msgChoice)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		ai.Logger.Task("backAsk", resp)
		return resp, nil
	}

	return ai.ResponseFailed(fmt.Sprintf("You execute unknown tool function in jira: %s", tc.Function.Name), resp)
}

func (ai *Ai) backAsk(id, content string, q mod.Message, assistantToolCallMsg mod.Message) (mod.ResponseBody, error) {
	toolMsg := mod.Message{
		Role:       "tool",
		ToolCallID: id,
		Content:    content,
	}
	messages := ai.Memory.HistoryMessage(q, ai.Config.PromtSystemChat)
	messages = append(messages, assistantToolCallMsg, toolMsg)
	resp, err := llmcalls.Ask(ai.makeBody(messages, GetToolRouter()), ai.Config)
	if err != nil {
		ai.Logger.Error("backAsk failed:", err)
		return mod.ResponseBody{}, err
	}
	msgChoice, err := firstChoice(resp)
	if err != nil {
		ai.Logger.Error("backAsk: firstChoice failed:", err)
		return mod.ResponseBody{}, err
	}
	ai.Memory.FillShortMemory(q.Content, msgChoice.Content)
	return resp, nil
}

func (ai *Ai) ResponseFailed(content string, resp mod.ResponseBody) (mod.ResponseBody, error) {
	failedstring := "Failed execute function: " + content
	msg := mod.Message{
		Role:    "user",
		Content: failedstring,
	}
	resp, err := llmcalls.Ask(ai.makeBody(ai.Memory.HistoryMessage(msg, ai.Config.PromtSystemChat), nil), ai.Config)
	if err != nil {
		ai.Logger.Error("ResponseFailed ask failed:", err)
		return mod.ResponseBody{}, err
	}
	msgChoice, err := firstChoice(resp)
	if err != nil {
		ai.Logger.Error("ResponseFailed: firstChoice failed:", err)
		return mod.ResponseBody{}, err
	}
	ai.Memory.FillShortMemory(failedstring, msgChoice.Content)
	return resp, nil
}

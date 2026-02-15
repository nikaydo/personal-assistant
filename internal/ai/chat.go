package ai

import (
	"encoding/json"
	"fmt"

	mod "github.com/nikaydo/jira-filler/internal/models"
)

type Memory struct {
	History []Question
	Count   int
}

type Question struct {
	Q            string
	A            string
	ContextToken int
}

func (ai *Ai) MakeAsk(q mod.Message, tools []mod.Tool) (mod.ResponseBody, error) {
	fmt.Println(ai.Memory.History)
	msg := ai.HistoryMessage(q)
	ai.Logger.Question(q)
	resp, err := ai.Ask(msg, tools)
	if err != nil {
		ai.Logger.Error(fmt.Sprintf("ask request failed: %v", err))
		return mod.ResponseBody{}, err
	}
	if resp.Error.Message != "" {
		ai.Logger.Error(resp.Error)
		return mod.ResponseBody{}, fmt.Errorf("error: %s", resp.Error.Message)
	}

	msgChoice, err := firstChoice(resp)
	if err != nil {
		ai.Logger.Error(err.Error())
		return mod.ResponseBody{}, err
	}

	if len(msgChoice.ToolCalls) > 0 {
		if msgChoice.Content != "" {
			ai.Memory.History = append(ai.Memory.History, Question{Q: q.Content, A: msgChoice.Content, ContextToken: resp.Usage.TotalTokens})
		}
		ai.Logger.Task("isTool", resp)
		resp, err := ai.isTool(resp, q)
		if err != nil {
			ai.Logger.Error(fmt.Sprintf("tool execution failed: %v", err))
			return resp, err
		}
		ai.Logger.Task("isTool Out", resp)
		ai.Logger.Answer(resp)
		return resp, nil
	}

	if msgChoice.Content == "" {
		err := fmt.Errorf("empty model answer without tool_calls")
		ai.Logger.Error(err.Error())
		return mod.ResponseBody{}, err
	}
	ai.Memory.History = append(ai.Memory.History, Question{Q: q.Content, A: msgChoice.Content, ContextToken: resp.Usage.TotalTokens})
	ai.Logger.Answer(resp)
	return resp, nil
}

func (ai *Ai) isTool(resp mod.ResponseBody, q mod.Message) (mod.ResponseBody, error) {
	tc, err := firstToolCall(resp)
	if err != nil {
		ai.Logger.Error(err.Error())
		return mod.ResponseBody{}, err
	}

	f := mod.ToolFunctionParseResponse{Name: tc.Function.Name}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &f); err != nil {
		ai.Logger.Error(fmt.Sprintf("tool args parse failed for %s: %v", tc.Function.Name, err))
		return mod.ResponseBody{}, err
	}
	switch f.Name {
	case "select_tool_group":
		switch f.Group {
		case "jira":
			resp, err := ai.JiraTasks(q)
			if err != nil {
				return mod.ResponseBody{}, err
			}
			return resp, nil
		}
	}

	msgChoice, err := firstChoice(resp)
	if err != nil {
		ai.Logger.Error(err.Error())
		return mod.ResponseBody{}, err
	}
	if msgChoice.Content != "" {
		return resp, nil
	}

	err = fmt.Errorf("unknown tool function: name=%s group=%s", f.Name, f.Group)
	ai.Logger.Error(err.Error())
	return mod.ResponseBody{}, err
}

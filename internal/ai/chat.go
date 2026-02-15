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
		return mod.ResponseBody{}, err
	}
	if resp.Error.Message != "" {
		ai.Logger.Error(resp.Error)
		return mod.ResponseBody{}, fmt.Errorf("error: %s", resp.Error.Message)
	}

	if len(resp.Choices[0].Message.ToolCalls) > 0 {
		if resp.Choices[0].Message.Content != "" {
			ai.Memory.History = append(ai.Memory.History, Question{Q: q.Content, A: resp.Choices[0].Message.Content, ContextToken: resp.Usage.TotalTokens})
		}
		ai.Logger.Task("isTool", resp)
		resp, err := ai.isTool(resp, q)
		if err != nil {
			return resp, err
		}
		ai.Logger.Task("isTool Out", resp)
		ai.Logger.Answer(resp)
		return resp, nil
	}

	ai.Logger.Answer(resp)
	return resp, nil
}

func (ai *Ai) isTool(resp mod.ResponseBody, q mod.Message) (mod.ResponseBody, error) {
	f := mod.ToolFunctionParseResponse{Name: resp.Choices[0].Message.ToolCalls[0].Function.Name}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.ToolCalls[0].Function.Arguments), &f); err != nil {
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
	if resp.Choices[0].Message.Content != "" {
		return resp, nil
	}
	return mod.ResponseBody{}, fmt.Errorf("unknown tool function: %v", resp)
}

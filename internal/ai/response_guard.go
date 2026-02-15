package ai

import (
	"fmt"

	mod "github.com/nikaydo/jira-filler/internal/models"
)

func firstChoice(resp mod.ResponseBody) (mod.Message, error) {
	if len(resp.Choices) == 0 {
		return mod.Message{}, fmt.Errorf("empty model response: choices is empty")
	}
	return resp.Choices[0].Message, nil
}

func firstToolCall(resp mod.ResponseBody) (mod.ToolCall, error) {

	msg, err := firstChoice(resp)
	if err != nil {
		return mod.ToolCall{}, err
	}
	if len(msg.ToolCalls) == 0 {
		return mod.ToolCall{}, fmt.Errorf("model did not return any tool_calls")
	}
	return msg.ToolCalls[0], nil
}

package ai

import (
	"fmt"

	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	mod "github.com/nikaydo/personal-assistant/internal/models"
)

func (ai *Ai) MakeAsk(q string, tools []mod.Tool) (mod.ResponseBody, error) {
	ai.Logger.Question(q)
	history := ai.Memory.MessageWithHistory(q, ai.Config.PromtSystemChat)
	ai.Logger.Info(
		"MakeAsk: sending LLM request",
		"history_messages", len(history),
		"tools_count", len(tools),
	)
	respLLM, err := ai.Queue.AddToQueue(llmcalls.QueueItem{Body: ai.makeBody(history, tools)})
	if err != nil {
		ai.Logger.Error("MakeAsk: ask request failed:", err)
		return mod.ResponseBody{}, err
	}
	if respLLM.Error.Message != "" {
		ai.Logger.Error(respLLM.Error)
		return mod.ResponseBody{}, fmt.Errorf("error: %s", respLLM.Error.Message)
	}

	msgChoice, err := firstChoice(respLLM)
	if err != nil {
		ai.Logger.Error("MakeAsk: firstChoice failed:", err)
		return mod.ResponseBody{}, err
	}

	if len(msgChoice.ToolCalls) > 0 {
		ai.Logger.Info("MakeAsk: model requested tool call", "tool_calls_count", len(msgChoice.ToolCalls))
		// if msgChoice.Content != "" {
		// 	ai.Memory.FillShortMemory(q.Content, msgChoice.Content)
		// }
		ai.Logger.Task("isTool", respLLM)
		resp, err := ai.isTool(respLLM)
		if err != nil {
			ai.Logger.Error("MakeAsk: tool execution failed:", err)
			return resp, err
		}
		ai.Logger.Answer(resp)
		return resp, nil
	}

	if msgChoice.Content == "" {
		err := fmt.Errorf("empty model answer without tool_calls")
		ai.Logger.Error("MakeAsk:", err)
		return mod.ResponseBody{}, err
	}
	ai.Memory.Memory(q, respLLM)
	ai.Logger.Answer(respLLM)
	return respLLM, nil
}

func (ai *Ai) isTool(resp mod.ResponseBody) (mod.ResponseBody, error) {
	tc, err := firstToolCall(resp)
	if err != nil {
		ai.Logger.Error("isTool: firstToolCall failed:", err)
		return mod.ResponseBody{}, err
	}

	f := mod.ToolFunctionParseResponse{Name: tc.Function.Name}
	if err := parseToolArguments(tc.Function.Arguments, &f); err != nil {
		ai.Logger.Error("isTool: tool args parse failed for", tc.Function.Name, ":", err)
		return mod.ResponseBody{}, err
	}
	switch f.Name {
	case "":

	}

	msgChoice, err := firstChoice(resp)
	if err != nil {
		ai.Logger.Error("isTool: firstChoice failed:", err)
		return mod.ResponseBody{}, err
	}
	if msgChoice.Content != "" {
		return resp, nil
	}

	err = fmt.Errorf("unknown tool function: name=%s group=%s", f.Name, f.Group)
	ai.Logger.Error("isTool:", err)
	return mod.ResponseBody{}, err
}

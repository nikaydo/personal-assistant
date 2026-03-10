package ai

import (
	"fmt"

	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	mod "github.com/nikaydo/personal-assistant/internal/models"
)

var addToQueueFn = func(q *llmcalls.Queue, item llmcalls.QueueItem) (mod.ResponseBody, error) {
	return q.AddToQueue(item)
}

func (ai *Ai) MakeAsk(q string, tools []mod.Tool) (mod.ResponseBody, error) {
	ai.Logger.Question(q)
	history := ai.Memory.MessageWithHistory(q, ai.Config.PromtSystemChat)
	ai.Logger.Info(
		"MakeAsk: sending LLM request",
		"history_messages", len(history),
		"tools_count", len(tools),
	)
	respLLM, err := addToQueueFn(ai.Queue, llmcalls.QueueItem{Body: ai.makeBody(history, tools)})
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
		ai.Logger.Task("Found tool in response", respLLM)
		// pass pointer to the system memory field so it can be created/updated
		resp, err := ai.Agent.DetectChosenTool(respLLM, ai.Memory.SystemMemory, ai.Memory.ToolsMemory, history)
		if err != nil {
			return mod.ResponseBody{}, err
		}
		ai.Logger.Task("DetectChosenTool:", respLLM, "Response:", resp)
		return resp, nil
	}

	if msgChoice.Content == "" {
		err := fmt.Errorf("empty model answer without tool_calls")
		ai.Logger.Error("MakeAsk:", err)
		return mod.ResponseBody{}, err
	}
	go func() {
		ai.Memory.Memory(q, respLLM, ai.Queue, ai.Model[0])
	}()

	ai.Logger.Answer(respLLM)
	return respLLM, nil
}

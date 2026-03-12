package ai

import (
	"encoding/json"
	"errors"
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
		ai.Logger.Task("Found tool in response, handling tool calls in chat flow", respLLM)
		firstTool := msgChoice.ToolCalls[0]
		if firstTool.Function.Name == "agent_mode" {
			*ai.Agent.History = []mod.Message{}
			var payload struct {
				Question string `json:"question"`
			}
			_ = json.Unmarshal([]byte(firstTool.Function.Arguments), &payload)
			if payload.Question == "" && q != "" {
				*ai.Agent.History = append(*ai.Agent.History, mod.Message{Role: "user", Content: q})
			}
		}
		respLLM, err = ai.Agent.DetectChosenTool(respLLM, ai.Memory.SystemMemory, ai.Memory.ToolsMemory, history)
		if err != nil {
			ai.Logger.Error("MakeAsk: tool call handling failed:", err)
			return mod.ResponseBody{}, err
		}
		if len(respLLM.Choices) == 0 {
			return mod.ResponseBody{}, errors.New("tool call produced empty response")
		}
		msgChoice, err = firstChoice(respLLM)
		if err != nil {
			ai.Logger.Error("MakeAsk: firstChoice failed after tool handling:", err)
			return mod.ResponseBody{}, err
		}
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

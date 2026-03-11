package ai

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nikaydo/personal-assistant/internal/agent"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	mod "github.com/nikaydo/personal-assistant/internal/models"
)

var addToQueueFn = func(q *llmcalls.Queue, item llmcalls.QueueItem) (mod.ResponseBody, error) {
	return q.AddToQueue(item)
}

var detectChosenToolFn = func(a *agent.Agent, body mod.ResponseBody, system *mod.SystemSettings, tools *[]mod.ToolsHistory, msg []mod.Message) (mod.ResponseBody, error) {
	return a.DetectChosenTool(body, system, tools, msg)
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
		ai.Logger.Task("Found tool in response, handling tool calls in chat flow", respLLM)

		// ensure agent is wired up with the current runtime context
		if ai.Agent.Queue == nil {
			ai.Agent.Queue = ai.Queue
		}
		if ai.Agent.Logger == nil {
			ai.Agent.Logger = ai.Logger
		}
		if ai.Agent.History == nil {
			ai.Agent.History = &[]mod.Message{}
		}
		if ai.Agent.Model == "" && len(ai.Model) > 0 {
			ai.Agent.Model = ai.Model[0]
		}
		if ai.Agent.SystemPrompt == "" {
			ai.Agent.SystemPrompt = ai.Config.PromtSystemAgent
		}
		if ai.Agent.Cfg.ApiKeyOpenrouter == "" {
			ai.Agent.Cfg = ai.Config
		}
		if ai.Agent.Dbase == nil {
			ai.Agent.Dbase = ai.Memory.DBase
		}

		// seed agent history with the question when agent_mode doesn't include it
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

		respLLM, err = detectChosenToolFn(&ai.Agent, respLLM, ai.Memory.SystemMemory, ai.Memory.ToolsMemory, history)
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

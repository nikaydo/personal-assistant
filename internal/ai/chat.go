package ai

import (
	"encoding/json"
	"fmt"

	"github.com/nikaydo/personal-assistant/internal/ai/memory"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	mod "github.com/nikaydo/personal-assistant/internal/models"
)

func (ai *Ai) MakeAsk(q mod.Message, tools []mod.Tool) (mod.ResponseBody, error) {
	ai.Logger.Question(q)
	history := ai.Memory.HistoryMessage(q, ai.Config.PromtSystemChat)
	ai.Logger.Info(
		"MakeAsk: sending LLM request",
		"history_messages", len(history),
		"tools_count", len(tools),
	)
	resp, err := llmcalls.Ask(ai.makeBody(history, tools), ai.Config)
	if err != nil {
		ai.Logger.Error("MakeAsk: ask request failed:", err)
		return mod.ResponseBody{}, err
	}
	if resp.Error.Message != "" {
		ai.Logger.Error(resp.Error)
		return mod.ResponseBody{}, fmt.Errorf("error: %s", resp.Error.Message)
	}

	msgChoice, err := firstChoice(resp)
	if err != nil {
		ai.Logger.Error("MakeAsk: firstChoice failed:", err)
		return mod.ResponseBody{}, err
	}

	if len(msgChoice.ToolCalls) > 0 {
		ai.Logger.Info("MakeAsk: model requested tool call", "tool_calls_count", len(msgChoice.ToolCalls))
		if msgChoice.Content != "" {
			ai.Memory.FillShortMemory(q.Content, msgChoice.Content)
		}
		ai.Logger.Task("isTool", resp)
		resp, err := ai.isTool(resp, q)
		if err != nil {
			ai.Logger.Error("MakeAsk: tool execution failed:", err)
			return resp, err
		}
		ai.Logger.Task("isTool Out", resp)
		ai.Logger.Answer(resp)
		return resp, nil
	}

	if msgChoice.Content == "" {
		err := fmt.Errorf("empty model answer without tool_calls")
		ai.Logger.Error("MakeAsk:", err)
		return mod.ResponseBody{}, err
	}
	ai.Memory.FillShortMemory(q.Content, msgChoice.Content)
	ai.Logger.Info("MakeAsk: summary step check", "short_memory_len", ai.Memory.ShortTermLen(), "summary_counter", ai.Memory.SummaryCounter())
	if run, snapshot := ai.Memory.PlanSummaryRun(); run {
		go func(data []memory.ShortTerm) {
			defer ai.Memory.FinishSummaryRun()
			if err := ai.Memory.SummMemoryFromSnapshot(data); err != nil {
				ai.Logger.Error("SummMemoryFromSnapshot:", err)
			}
		}(snapshot)
	}

	ai.Memory.CheckLimits()
	ai.Logger.Answer(resp)
	return resp, nil
}

func (ai *Ai) isTool(resp mod.ResponseBody, q mod.Message) (mod.ResponseBody, error) {
	tc, err := firstToolCall(resp)
	if err != nil {
		ai.Logger.Error("isTool: firstToolCall failed:", err)
		return mod.ResponseBody{}, err
	}

	f := mod.ToolFunctionParseResponse{Name: tc.Function.Name}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &f); err != nil {
		ai.Logger.Error("isTool: tool args parse failed for", tc.Function.Name, ":", err)
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

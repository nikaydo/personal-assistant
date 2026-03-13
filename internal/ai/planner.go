package ai

import (
	"encoding/json"
	"strings"

	"github.com/nikaydo/personal-assistant/internal/ai/memory"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	mod "github.com/nikaydo/personal-assistant/internal/models"
)

type requestRoute string

const (
	routeDirect      requestRoute = "direct"
	routeAgent       requestRoute = "agent"
	routeMemoryHeavy requestRoute = "memory-heavy"
)

type requestPlan struct {
	Route  requestRoute
	Reason string
}

type plannerDecision struct {
	Route  string `json:"route"`
	Reason string `json:"reason"`
}

func (ai *Ai) planRequest(question string) requestPlan {
	if route, ok := heuristicRoute(question); ok {
		return requestPlan{Route: route, Reason: "heuristic"}
	}

	// Phase A: lightweight planner call without tools and without long-term retrieval.
	plannerSystem := "You route requests for a local assistant. Output only JSON: {\"route\":\"direct|agent|memory-heavy\",\"reason\":\"short\"}. Use agent for file/command/external actions."
	msg := ai.Memory.MessageWithHistoryWithOptions(question, plannerSystem, memory.BuildOptions{
		IncludeLongTerm:    false,
		IncludeToolsMemory: false,
	})
	resp, err := ai.Queue.AddToQueue(llmcalls.QueueItem{Body: ai.makeBody(msg, nil)})
	if err != nil {
		ai.Logger.Warn("planRequest: planner call failed, fallback to direct", "error", err)
		return requestPlan{Route: routeDirect, Reason: "planner_error_fallback"}
	}
	content := strings.TrimSpace(resp.GetContent())
	if content == "" {
		return requestPlan{Route: routeDirect, Reason: "planner_empty_fallback"}
	}
	decision, ok := parsePlannerDecision(content)
	if !ok {
		return requestPlan{Route: routeDirect, Reason: "planner_parse_fallback"}
	}
	switch decision.Route {
	case string(routeAgent):
		return requestPlan{Route: routeAgent, Reason: nonEmpty(decision.Reason, "planner")}
	case string(routeMemoryHeavy):
		return requestPlan{Route: routeMemoryHeavy, Reason: nonEmpty(decision.Reason, "planner")}
	default:
		return requestPlan{Route: routeDirect, Reason: nonEmpty(decision.Reason, "planner")}
	}
}

func heuristicRoute(question string) (requestRoute, bool) {
	q := strings.ToLower(strings.TrimSpace(question))
	if q == "" {
		return routeDirect, true
	}
	agentMarkers := []string{
		"create file", "delete file", "rename file", "run command", "terminal", "shell", "bash", "cmd",
		"создай файл", "удали файл", "переименуй файл", "выполни команду", "терминал", "консоль", "команду",
	}
	for _, marker := range agentMarkers {
		if strings.Contains(q, marker) {
			return routeAgent, true
		}
	}
	return routeDirect, false
}

func parsePlannerDecision(content string) (plannerDecision, bool) {
	raw := content
	if extracted, err := mod.ExtractJSON(content); err == nil {
		raw = extracted
	}
	var d plannerDecision
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return plannerDecision{}, false
	}
	d.Route = strings.TrimSpace(strings.ToLower(d.Route))
	d.Reason = strings.TrimSpace(d.Reason)
	return d, d.Route != ""
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

package agent

import (
	"encoding/json"
	"strings"
	"testing"

	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

func mustAgentResponse(t *testing.T, s string) AgentResponse {
	t.Helper()
	var r AgentResponse
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		t.Fatalf("failed to unmarshal agent response: %v", err)
	}
	return r
}

func TestGetAgentTool_IncludesCommand(t *testing.T) {
	tools := GetAgentTool()
	found := false
	for _, tl := range tools {
		if tl.Function.Name == "command" {
			found = true
			params, ok := tl.Function.Parameters.(map[string]any)
			if !ok {
				t.Fatalf("parameters not map")
			}
			props := params["properties"].(map[string]any)
			if _, ok := props["command"]; !ok {
				t.Errorf("command property missing")
			}
			break
		}
	}
	if !found {
		t.Error("command tool not present")
	}
}

func fakeResponse(fn string, args string) models.ResponseBody {
	return models.ResponseBody{
		Choices: []models.Choices{{
			Message: models.Message{
				ToolCalls: []models.ToolCall{{
					Function: models.ToolFunction{Name: fn, Arguments: args},
				}},
			},
		}},
	}
}

func TestRunTool_CommandExec(t *testing.T) {
	agent := Agent{Logger: logg.InitLogger()}
	resp := fakeResponse("command", "pwd")
	out, stop, err := agent.RunTool(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stop {
		t.Fatal("stop flag should be false")
	}
	if out == "" {
		t.Error("expected non-empty output from pwd")
	}
}

func TestRunTool_CommandJSON(t *testing.T) {
	agent := Agent{Logger: logg.InitLogger()}
	payload := map[string]any{"command": "pwd", "args": []string{}}
	b, _ := json.Marshal(payload)
	resp := fakeResponse("command", string(b))
	out, _, err := agent.RunTool(resp)
	if err != nil {
		t.Fatalf("error running command: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output for json form")
	}
}

func TestRunTool_ReasoningActionArguments(t *testing.T) {
	agent := Agent{Logger: logg.InitLogger()}
	resp := fakeResponse("reasoning", `{"thought":"t","action":{"function":"command","arguments":{"command":"echo","args":["hi"]}}}`)
	out, stop, err := agent.RunTool(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stop {
		t.Fatal("stop flag should be false")
	}
	if !strings.Contains(out, "hi") {
		t.Errorf("expected output to contain command result, got %q", out)
	}
}

func TestRunTool_ReasoningActionArgsAlias(t *testing.T) {
	agent := Agent{Logger: logg.InitLogger()}
	resp := fakeResponse("reasoning", `{"thought":"t","action":{"function":"command","args":["echo","hi"]}}`)
	out, stop, err := agent.RunTool(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stop {
		t.Fatal("stop flag should be false")
	}
	if !strings.Contains(out, "hi") {
		t.Errorf("expected output to contain command result, got %q", out)
	}
}

func TestNormalizeCommandSpec_ArgumentsArray(t *testing.T) {
	r := mustAgentResponse(t, `{"thought":"t","action":{"function":"command","arguments":["echo","ok"]}}`)
	spec, err := normalizeCommandSpec(r)
	if err != nil {
		t.Fatalf("normalizeCommandSpec failed: %v", err)
	}
	if spec.Command != "echo" || len(spec.Args) != 1 || spec.Args[0] != "ok" || spec.Mode != "exec" {
		t.Fatalf("unexpected spec: %+v", spec)
	}
}

func TestNormalizeCommandSpec_ArgsObject(t *testing.T) {
	r := mustAgentResponse(t, `{"thought":"t","action":{"function":"command","args":{"command":"echo","args":["ok"],"mode":"exec"}}}`)
	spec, err := normalizeCommandSpec(r)
	if err != nil {
		t.Fatalf("normalizeCommandSpec failed: %v", err)
	}
	if spec.Command != "echo" || len(spec.Args) != 1 || spec.Args[0] != "ok" || spec.Mode != "exec" {
		t.Fatalf("unexpected spec: %+v", spec)
	}
}

func TestNormalizeCommandSpec_FunctionWithPrefix(t *testing.T) {
	r := mustAgentResponse(t, `{"thought":"t","action":{"function":"functions.command","arguments":{"command":"echo","args":["ok"]}}}`)
	spec, err := normalizeCommandSpec(r)
	if err != nil {
		t.Fatalf("normalizeCommandSpec failed: %v", err)
	}
	if spec.Command != "echo" || len(spec.Args) != 1 || spec.Args[0] != "ok" {
		t.Fatalf("unexpected spec: %+v", spec)
	}
}

func TestNormalizeCommandSpec_InfersCommandWhenMissingFunction(t *testing.T) {
	r := mustAgentResponse(t, `{"thought":"t","action":{"arguments":{"command":"echo","args":["ok"]}}}`)
	spec, err := normalizeCommandSpec(r)
	if err != nil {
		t.Fatalf("normalizeCommandSpec failed: %v", err)
	}
	if spec.Command != "echo" || len(spec.Args) != 1 || spec.Args[0] != "ok" {
		t.Fatalf("unexpected spec: %+v", spec)
	}
}

func TestNormalizeCommandSpec_Conflict(t *testing.T) {
	r := mustAgentResponse(t, `{"thought":"t","action":{"function":"command","arguments":{"command":"echo"},"args":["echo","ok"]}}`)
	if _, err := normalizeCommandSpec(r); err == nil {
		t.Fatal("expected conflict error when both arguments and args are provided")
	}
}

func TestNormalizeCommandSpec_ShellArrayRejected(t *testing.T) {
	r := mustAgentResponse(t, `{"thought":"t","action":{"function":"command","mode":"shell","arguments":["echo","ok"]}}`)
	if _, err := normalizeCommandSpec(r); err == nil {
		t.Fatal("expected shell array payload to be rejected")
	}
}

func TestNormalizeCommandSpec_ShellArraySingleAccepted(t *testing.T) {
	r := mustAgentResponse(t, `{"thought":"t","action":{"function":"command","mode":"shell","arguments":["echo ok > tts.txt"]}}`)
	spec, err := normalizeCommandSpec(r)
	if err != nil {
		t.Fatalf("normalizeCommandSpec failed: %v", err)
	}
	if spec.Mode != "shell" || spec.Command != "echo ok > tts.txt" {
		t.Fatalf("unexpected spec: %+v", spec)
	}
}

func TestRunTool_ReasoningInvalidPayloadReturnsToolResult(t *testing.T) {
	agent := Agent{Logger: logg.InitLogger()}
	resp := fakeResponse("reasoning", `{"thought":"t","action":{"function":"command","mode":"shell","arguments":["echo","oops"]}}`)
	out, stop, err := agent.RunTool(resp)
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}
	if stop {
		t.Fatal("stop flag should be false")
	}
	tr, ok := toolResultFromOutput(out)
	if !ok {
		t.Fatalf("expected tool result json, got: %q", out)
	}
	if tr.Ok {
		t.Fatalf("expected failed tool result, got: %+v", tr)
	}
	if !tr.Retryable {
		t.Fatalf("expected retryable=true, got: %+v", tr)
	}
}

func TestFailureGuard_RepeatedFailureStops(t *testing.T) {
	g := newFailureGuard(2)
	if g.observe("a", "err", true) {
		t.Fatal("must not stop on first failure")
	}
	if !g.observe("a", "err", true) {
		t.Fatal("must stop on second identical failure")
	}
}

func TestFailureGuard_ResetsOnSuccess(t *testing.T) {
	g := newFailureGuard(2)
	_ = g.observe("a", "err", true)
	if g.observe("", "", false) {
		t.Fatal("success should reset guard")
	}
	if g.observe("a", "err", true) {
		t.Fatal("must not stop after reset on first failure")
	}
}

func TestHistorySanitization_EmptyThoughts(t *testing.T) {
	agent := Agent{Logger: logg.InitLogger(), History: &[]models.Message{}}

	// initial thought (empty) should be replaced, and CollectContext should likewise
	r := AgentResponse{Thought: ""}
	// mimic initial append logic
	safeThought := r.Thought
	if safeThought == "" {
		safeThought = " "
	}
	*agent.History = append(*agent.History, models.Message{Role: "assistant", Content: safeThought})
	if (*agent.History)[0].Content == "" {
		t.Errorf("initial thought was not sanitized")
	}

	// now test CollectContext with an empty thought
	body := fakeResponse("foo", `{"thought":""}`)
	// also need to simulate a tool call structure expected by CollectContext
	body.Choices[0].Message.ToolCalls = []models.ToolCall{{
		Function: models.ToolFunction{Name: "foo", Arguments: "{}"},
		ID:       "id",
		Type:     "type",
	}}
	// run CollectContext
	if err := agent.CollectContext(body, "output"); err != nil {
		t.Fatalf("CollectContext error: %v", err)
	}
	last := (*agent.History)[len(*agent.History)-1]
	if last.Content == "" {
		t.Errorf("CollectContext appended empty content")
	}
}

func TestCollectContext_IncludesOutput(t *testing.T) {
	agent := Agent{Logger: logg.InitLogger(), History: &[]models.Message{}}
	body := fakeResponse("command", "{\"command\":\"pwd\"}")
	body.Choices[0].Message.ToolCalls[0].ID = "x"
	body.Choices[0].Message.ToolCalls[0].Type = "cmd"

	if err := agent.CollectContext(body, "dir123"); err != nil {
		t.Fatalf("CollectContext failed: %v", err)
	}
	if len(*agent.History) < 1 {
		t.Fatalf("history too short: %v", *agent.History)
	}
	funcMsg := (*agent.History)[len(*agent.History)-1]
	if !containsSubstring(funcMsg.Content, "dir123") {
		t.Errorf("expected output in function message content, got %q", funcMsg.Content)
	}
}

// verify that a non-empty SystemPrompt is prepended to the history when
// AskLLM is called.  We don't need the queue to actually succeed; an empty
// , unstarted queue will return an error, but the history mutation happens
// before the call, so we can ignore the result.
func TestAskLLM_AddsSystemPrompt(t *testing.T) {
	logger := logg.InitLogger()
	agent := Agent{Logger: logger, History: &[]models.Message{}, Model: "m", SystemPrompt: "agent-prompt"}

	// use a minimal queue instance; it does not need to be started
	agent.Queue = &llmcalls.Queue{}

	// call AskLLM and intentionally ignore the error
	_, _ = agent.AskLLM("auto")

	if len(*agent.History) == 0 {
		t.Fatal("history empty after AskLLM")
	}
	if (*agent.History)[0].Role != "system" || (*agent.History)[0].Content != "agent-prompt" {
		t.Errorf("first history entry wrong: %+v", (*agent.History)[0])
	}
}

// helper for the previous test
func containsSubstring(s, sub string) bool {
	return strings.Contains(s, sub)
}

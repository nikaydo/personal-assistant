package agent

import (
	"encoding/json"
	"testing"

	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

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

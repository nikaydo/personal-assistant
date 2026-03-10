package agent

import (
	"encoding/json"
	"testing"

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
	agent := Agent{}
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
	agent := Agent{}
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

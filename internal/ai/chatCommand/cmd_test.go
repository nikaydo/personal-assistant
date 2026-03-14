package chatcommand

import "testing"

func TestCheckCmd_WebSearch(t *testing.T) {
	param, ok := CheckCmd("/web latest postgres release")
	if !ok {
		t.Fatal("expected /web to be recognized")
	}
	if param.Message != "latest postgres release" {
		t.Fatalf("unexpected message: %q", param.Message)
	}
	if param.SystemPrompt == "" {
		t.Fatal("expected system prompt to be set")
	}
}

func TestCheckCmd_AgentStripsPrefix(t *testing.T) {
	param, ok := CheckCmd("/agent list project files")
	if !ok {
		t.Fatal("expected /agent to be recognized")
	}
	if len(param.Tool) == 0 {
		t.Fatal("expected agent tools to be present")
	}
	if param.Message != "list project files" {
		t.Fatalf("unexpected message: %q", param.Message)
	}
}

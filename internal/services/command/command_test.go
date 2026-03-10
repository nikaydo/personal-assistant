package command

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestService_ExecuteFromLLM_Simple(t *testing.T) {
	svc := NewService()
	out, err := svc.ExecuteFromLLM("pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Errorf("expected cwd output, got empty string")
	}
}

func TestService_ExecuteFromLLM_JSON(t *testing.T) {
	svc := NewService()
	// use "pwd" again; args slice should be empty
	req := map[string]any{"command": "pwd", "args": []string{}}
	b, _ := json.Marshal(req)
	out, err := svc.ExecuteFromLLM(string(b))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Errorf("expected cwd output, got empty string")
	}
}

func TestService_ExecuteFromLLM_NotAllowed(t *testing.T) {
	svc := NewService()
	_, err := svc.ExecuteFromLLM("some_unknown_command")
	if err == nil {
		t.Fatal("expected error for disallowed command")
	}
}

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

func TestService_ExecuteFromLLM_Blocked(t *testing.T) {
	// temporarily insert a simple blocked command and verify it is
	// rejected.  "echo" is safe to run, so we use it for this test.
	blocked["echo"] = struct{}{}
	defer delete(blocked, "echo")

	svc := NewService()
	_, err := svc.ExecuteFromLLM("echo hello")
	if err == nil {
		t.Fatal("expected error for blocked command")
	}
}

func TestService_ExecuteFromLLM_UnknownAllowed(t *testing.T) {
	// commands not on the blacklist should pass through, even if they
	// aren't explicitly blacklisted.  "true" is a trivial builtin.
	svc := NewService()
	out, err := svc.ExecuteFromLLM("true")
	if err != nil {
		t.Fatalf("unexpected error for allowed command: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty output from 'true', got %q", out)
	}
}

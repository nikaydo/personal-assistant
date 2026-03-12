package command

import "testing"

func TestExecuteSpec_ExecMode_MultilineArgStable(t *testing.T) {
	svc := NewService()
	spec := CommandSpec{
		Command: "printf",
		Args:    []string{"%s", "line1\nline2"},
		Mode:    "exec",
	}
	out := svc.ExecuteSpec(spec, CommandList{Type: false})
	if !out.Ok {
		t.Fatalf("expected ok result, got: %+v", out)
	}
	if out.Stdout != "line1\nline2" {
		t.Fatalf("unexpected stdout: %q", out.Stdout)
	}
}

func TestExecuteSpec_ShellMode_Explicit(t *testing.T) {
	svc := NewService()
	spec := CommandSpec{
		Command: "printf '%s' shell-ok",
		Mode:    "shell",
	}
	out := svc.ExecuteSpec(spec, CommandList{Type: false})
	if !out.Ok {
		t.Fatalf("expected ok result, got: %+v", out)
	}
	if out.Stdout != "shell-ok" {
		t.Fatalf("unexpected stdout: %q", out.Stdout)
	}
}

func TestExecuteSpec_ExitCodeFailureSetsError(t *testing.T) {
	svc := NewService()
	spec := CommandSpec{
		Command: "false",
		Mode:    "exec",
	}
	out := svc.ExecuteSpec(spec, CommandList{Type: false})
	if out.Ok {
		t.Fatalf("expected failure result, got: %+v", out)
	}
	if out.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code, got: %+v", out)
	}
	if out.Error == "" {
		t.Fatalf("expected non-empty error, got: %+v", out)
	}
}

func TestExecuteSpec_ShellCatAppendWithoutInputRejected(t *testing.T) {
	svc := NewService()
	spec := CommandSpec{
		Command: "cat >> tts.txt",
		Mode:    "shell",
	}
	out := svc.ExecuteSpec(spec, CommandList{Type: false})
	if out.Ok {
		t.Fatalf("expected shell validation failure, got: %+v", out)
	}
	if out.Error == "" {
		t.Fatalf("expected non-empty error, got: %+v", out)
	}
}

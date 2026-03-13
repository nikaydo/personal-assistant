package command

import (
	"strings"
	"testing"
)

func newTestService(list CommandList) *Service {
	return &Service{
		cmd:     &Command{},
		CmdList: list,
	}
}

func TestExecuteSpec_ExecMode_MultilineArgStable(t *testing.T) {
	svc := newTestService(CommandList{Type: false, List: map[string][]string{}})
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
	svc := newTestService(CommandList{Type: false, List: map[string][]string{}})
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
	svc := newTestService(CommandList{Type: false, List: map[string][]string{}})
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

func TestExecuteSpec_RepairsExecCommandWithFlags(t *testing.T) {
	svc := newTestService(CommandList{Type: false, List: map[string][]string{}})
	out := svc.ExecuteSpec(CommandSpec{
		Command: "printf %s",
		Args:    []string{"ok"},
		Mode:    "exec",
	}, CommandList{Type: false})
	if !out.Ok {
		t.Fatalf("expected repaired exec command to succeed, got: %+v", out)
	}
	if out.Stdout != "ok" {
		t.Fatalf("unexpected stdout: %q", out.Stdout)
	}
}

func TestExecuteSpec_ShellStderrIsFailure(t *testing.T) {
	svc := newTestService(CommandList{Type: false, List: map[string][]string{}})
	out := svc.ExecuteSpec(CommandSpec{
		Command: "printf ok >&2",
		Mode:    "shell",
	}, CommandList{Type: false})
	if out.Ok {
		t.Fatalf("expected shell stderr to be treated as failure, got: %+v", out)
	}
	if out.Error == "" {
		t.Fatalf("expected error from shell stderr, got: %+v", out)
	}
	if !out.Retryable {
		t.Fatalf("expected retryable shell failure, got: %+v", out)
	}
}

func TestExecuteSpec_ShellCatAppendWithoutInputRejected(t *testing.T) {
	svc := newTestService(CommandList{Type: false, List: map[string][]string{}})
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

func TestExecuteSpec_ShellMultipleCommandsAllowed(t *testing.T) {
	svc := newTestService(CommandList{
		Type: false,
		List: map[string][]string{"rm": nil},
	})
	out := svc.ExecuteSpec(CommandSpec{
		Command: "echo a; printf b",
		Mode:    "shell",
	}, CommandList{Type: false})
	if !out.Ok {
		t.Fatalf("expected shell chain to pass, got: %+v", out)
	}
	if out.Stdout != "a\nb" {
		t.Fatalf("unexpected stdout: %q", out.Stdout)
	}
}

func TestExecuteSpec_ShellBlockedCommandInChain(t *testing.T) {
	svc := newTestService(CommandList{
		Type: false,
		List: map[string][]string{"rm": nil},
	})
	out := svc.ExecuteSpec(CommandSpec{
		Command: "echo ok && rm -rf x && echo done",
		Mode:    "shell",
	}, CommandList{Type: false})
	if out.Ok {
		t.Fatalf("expected blocked command in chain, got: %+v", out)
	}
	if out.Error != "Command not allowed" {
		t.Fatalf("unexpected error: %q", out.Error)
	}
	if out.Retryable {
		t.Fatalf("expected non-retryable policy rejection, got: %+v", out)
	}
}

func TestExecuteSpec_ShellPipeValidatesEachCommand(t *testing.T) {
	svc := newTestService(CommandList{
		Type: false,
		List: map[string][]string{"grep": nil},
	})
	out := svc.ExecuteSpec(CommandSpec{
		Command: "echo a | grep a",
		Mode:    "shell",
	}, CommandList{Type: false})
	if out.Ok {
		t.Fatalf("expected blocked piped command, got: %+v", out)
	}
	if out.Error != "Command not allowed" {
		t.Fatalf("unexpected error: %q", out.Error)
	}
}

func TestExecuteSpec_ShellOrValidatesEachCommand(t *testing.T) {
	svc := newTestService(CommandList{
		Type: false,
		List: map[string][]string{"rm": nil},
	})
	out := svc.ExecuteSpec(CommandSpec{
		Command: "false || rm x",
		Mode:    "shell",
	}, CommandList{Type: false})
	if out.Ok {
		t.Fatalf("expected blocked || command, got: %+v", out)
	}
	if out.Error != "Command not allowed" {
		t.Fatalf("unexpected error: %q", out.Error)
	}
}

func TestExecuteSpec_ShellNewlineValidatesEachCommand(t *testing.T) {
	svc := newTestService(CommandList{
		Type: false,
		List: map[string][]string{"rm": nil},
	})
	out := svc.ExecuteSpec(CommandSpec{
		Command: "echo a\necho b",
		Mode:    "shell",
	}, CommandList{Type: false})
	if !out.Ok {
		t.Fatalf("expected newline-separated commands to pass, got: %+v", out)
	}
	if out.Stdout != "a\nb\n" {
		t.Fatalf("unexpected stdout: %q", out.Stdout)
	}
}

func TestExecuteSpec_ShellBlockedConstruct_CommandSubstitution(t *testing.T) {
	svc := newTestService(CommandList{Type: false, List: map[string][]string{}})
	out := svc.ExecuteSpec(CommandSpec{
		Command: "echo $(whoami)",
		Mode:    "shell",
	}, CommandList{Type: false})
	if out.Ok {
		t.Fatalf("expected blocked command substitution, got: %+v", out)
	}
	if !strings.Contains(out.Error, "blocked shell construct") {
		t.Fatalf("expected blocked shell construct error, got: %q", out.Error)
	}
}

func TestExecuteSpec_ShellBlockedConstruct_Backticks(t *testing.T) {
	svc := newTestService(CommandList{Type: false, List: map[string][]string{}})
	out := svc.ExecuteSpec(CommandSpec{
		Command: "echo `whoami`",
		Mode:    "shell",
	}, CommandList{Type: false})
	if out.Ok {
		t.Fatalf("expected blocked backticks, got: %+v", out)
	}
	if !strings.Contains(out.Error, "blocked shell construct") {
		t.Fatalf("expected blocked shell construct error, got: %q", out.Error)
	}
}

func TestExecuteSpec_ShellBlockedConstruct_Subshell(t *testing.T) {
	svc := newTestService(CommandList{Type: false, List: map[string][]string{}})
	out := svc.ExecuteSpec(CommandSpec{
		Command: "(echo hi)",
		Mode:    "shell",
	}, CommandList{Type: false})
	if out.Ok {
		t.Fatalf("expected blocked subshell, got: %+v", out)
	}
	if !strings.Contains(out.Error, "blocked shell construct") {
		t.Fatalf("expected blocked shell construct error, got: %q", out.Error)
	}
}

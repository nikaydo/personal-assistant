package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/shlex"
)

// Service provides a thin facade around command execution.  It is
// targeted at inputs produced by the language model; the argument block
// that the model supplies can be either a simple shell-like command
// line or a small JSON document with explicit fields.  The service is
// responsible for parsing the input, validating it against the
// blacklist and invoking the lower-level Command object.

// Service holds state that is shared between invocations, currently only
// the working directory as modified by "cd" commands.
type Service struct {
	cmd     *Command
	CmdList CommandList
}

// type if true then its allowed list if false its blocklist
type CommandList struct {
	Type bool                `json:"type"`
	List map[string][]string `json:"commands"`
}

type CommandSpec struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Mode    string   `json:"mode,omitempty"`
}

type ToolResult struct {
	Ok        bool   `json:"ok"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  int    `json:"exit_code"`
	Error     string `json:"error"`
	Retryable bool   `json:"retryable"`
}

var catAppendWithoutInputRe = regexp.MustCompile(`(?m)^\s*cat\s*>>`)
var shellRedirectTargetRe = regexp.MustCompile(`(?:>>|>)\s*['"]?([^'" \n;|&]+)['"]?`)

func (s *Service) CheckCommand(cmdToExec string, args []string) bool {
	switch s.CmdList.Type {
	//allowed to execute
	//if command in List then command allowed to execute
	//but if argument not in list of arguments command decline
	case true:
		allowedArgs, ok := s.CmdList.List[cmdToExec]
		if ok {
			if allowedArgs == nil {
				return true
			}
			for _, i := range args {
				if !slices.Contains(allowedArgs, i) {
					return false
				}
			}
			return true
		}

		return false
	//denied to execute
	//if command not in list then allowed to execute
	case false:
		_, ok := s.CmdList.List[cmdToExec]
		if !ok {
			return true
		}
		return false
	default:
		return false
	}
}

// NewService returns a ready-to-use command service.
func NewService() (*Service, error) {
	svc := &Service{
		cmd: &Command{},
		CmdList: CommandList{
			Type: false,
			List: map[string][]string{},
		},
	}
	list, err := LoadCommandList()
	if err != nil {
		return svc, err
	}
	svc.CmdList = list
	return svc, nil
}

func LoadCommandList() (CommandList, error) {
	b, err := os.ReadFile("./data/command_list.json")
	if err != nil {
		return CommandList{}, err
	}
	var c CommandList
	if err := json.Unmarshal(b, &c); err != nil {
		return CommandList{}, err
	}
	return c, nil
}

func (s *Service) ExecuteFromLLM(raw string, cList CommandList) ToolResult {
	spec, err := s.parse(raw)
	if err != nil {
		return ToolResult{
			Ok:        false,
			Error:     err.Error(),
			Retryable: false,
		}
	}
	return s.ExecuteSpec(spec, cList)
}

func (s *Service) ExecuteSpec(spec CommandSpec, cList CommandList) ToolResult {
	result := ToolResult{
		Ok:       false,
		ExitCode: -1,
	}
	if s == nil || s.cmd == nil {
		result.Error = "command service is not initialized"
		return result
	}
	spec = normalizeExecSpec(spec)
	mode := normalizeMode(spec.Mode)
	if spec.Command == "" {
		result.Error = "empty command"
		return result
	}
	switch mode {
	case "exec":
		if !s.CheckCommand(spec.Command, spec.Args) {
			result.Error = "Command not allowed"
			return result
		}
		stdout, stderr, exitCode, err := s.cmd.Exec(spec.Command, spec.Args...)
		result.Stdout = stdout
		result.Stderr = stderr
		result.ExitCode = exitCode
		if err != nil {
			result.Error = err.Error()
			result.Retryable = false
			return result
		}
	case "shell":
		// shell mode is explicit and executes script via "sh -c".
		if err := s.validateAndAuthorizeShellScript(spec.Command); err != nil {
			result.Error = err.Error()
			result.Retryable = false
			return result
		}
		stdout, stderr, exitCode, err := s.cmd.ExecShell(spec.Command)
		result.Stdout = stdout
		result.Stderr = stderr
		result.ExitCode = exitCode
		if err != nil {
			result.Error = err.Error()
			result.Retryable = false
			return result
		}
		if strings.TrimSpace(stderr) != "" {
			result.Error = strings.TrimSpace(stderr)
			result.Retryable = true
			return result
		}
	default:
		result.Error = fmt.Sprintf("unsupported mode: %s", spec.Mode)
		return result
	}
	if result.ExitCode != 0 {
		if strings.TrimSpace(result.Stderr) != "" {
			result.Error = strings.TrimSpace(result.Stderr)
		} else {
			result.Error = fmt.Sprintf("command exited with code %d", result.ExitCode)
		}
		result.Retryable = true
		return result
	}
	result.Ok = true
	if mode == "shell" {
		if target, ok := extractShellRedirectTarget(spec.Command); ok {
			if _, err := os.Stat(target); err != nil {
				result.Ok = false
				result.Error = fmt.Sprintf("shell command finished but target file is missing: %s", target)
				result.Retryable = true
			}
		}
	}
	return result
}

func validateShellScript(script string) error {
	s := strings.TrimSpace(script)
	if s == "" {
		return errors.New("empty shell script")
	}
	if catAppendWithoutInputRe.MatchString(s) && !strings.Contains(s, "<<") {
		return errors.New("invalid shell script: 'cat >> file' requires input (use heredoc or printf)")
	}
	return nil
}

func (s *Service) validateAndAuthorizeShellScript(script string) error {
	if err := validateShellScript(script); err != nil {
		return err
	}
	commands, err := parseShellTopLevelCommands(script)
	if err != nil {
		return err
	}
	for _, cmd := range commands {
		spec := normalizeExecSpec(cmd)
		if spec.Command == "" {
			continue
		}
		if !s.CheckCommand(spec.Command, spec.Args) {
			return fmt.Errorf("Command not allowed")
		}
	}
	return nil
}

func parseShellTopLevelCommands(script string) ([]CommandSpec, error) {
	var (
		buf         strings.Builder
		commands    []CommandSpec
		inSingle    bool
		inDouble    bool
		escaped     bool
		scriptRunes = []rune(script)
	)
	flush := func() error {
		part := strings.TrimSpace(buf.String())
		buf.Reset()
		if part == "" {
			return nil
		}
		argv, err := shlex.Split(part)
		if err != nil {
			return fmt.Errorf("invalid shell segment %s: %w", strconv.Quote(part), err)
		}
		if len(argv) == 0 {
			return nil
		}
		commands = append(commands, CommandSpec{
			Command: argv[0],
			Args:    argv[1:],
			Mode:    "exec",
		})
		return nil
	}

	for i := 0; i < len(scriptRunes); i++ {
		ch := scriptRunes[i]
		next := rune(0)
		if i+1 < len(scriptRunes) {
			next = scriptRunes[i+1]
		}

		if escaped {
			buf.WriteRune(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			buf.WriteRune(ch)
			escaped = true
			continue
		}

		if !inSingle && ch == '"' {
			inDouble = !inDouble
			buf.WriteRune(ch)
			continue
		}
		if !inDouble && ch == '\'' {
			inSingle = !inSingle
			buf.WriteRune(ch)
			continue
		}

		if !inSingle && !inDouble {
			if ch == '`' {
				return nil, errors.New("blocked shell construct: backticks are not allowed")
			}
			if ch == '$' && next == '(' {
				return nil, errors.New("blocked shell construct: command substitution is not allowed")
			}
			if ch == '(' || ch == ')' {
				return nil, errors.New("blocked shell construct: subshell is not allowed")
			}
			if ch == ';' || ch == '\n' {
				if err := flush(); err != nil {
					return nil, err
				}
				continue
			}
			if ch == '&' && next == '&' {
				if err := flush(); err != nil {
					return nil, err
				}
				i++
				continue
			}
			if ch == '|' && next == '|' {
				if err := flush(); err != nil {
					return nil, err
				}
				i++
				continue
			}
			if ch == '|' {
				if err := flush(); err != nil {
					return nil, err
				}
				continue
			}
		}
		buf.WriteRune(ch)
	}

	if escaped || inSingle || inDouble {
		return nil, errors.New("invalid shell script: unterminated escape or quote")
	}
	if err := flush(); err != nil {
		return nil, err
	}
	if len(commands) == 0 {
		return nil, errors.New("empty shell script")
	}
	return commands, nil
}

func extractShellRedirectTarget(script string) (string, bool) {
	matches := shellRedirectTargetRe.FindAllStringSubmatch(script, -1)
	if len(matches) == 0 {
		return "", false
	}
	last := matches[len(matches)-1]
	if len(last) < 2 {
		return "", false
	}
	target := strings.TrimSpace(last[1])
	if target == "" {
		return "", false
	}
	return target, true
}

func normalizeMode(mode string) string {
	if strings.TrimSpace(mode) == "" {
		return "exec"
	}
	return strings.ToLower(strings.TrimSpace(mode))
}

func normalizeExecSpec(spec CommandSpec) CommandSpec {
	if normalizeMode(spec.Mode) != "exec" {
		return spec
	}
	cmd := strings.TrimSpace(spec.Command)
	if cmd == "" || !strings.ContainsAny(cmd, " \t") {
		return spec
	}
	parts, err := shlex.Split(cmd)
	if err != nil || len(parts) == 0 {
		return spec
	}
	spec.Command = parts[0]
	spec.Args = append(parts[1:], spec.Args...)
	return spec
}

// parse decodes the raw input into a command name and argument slice.
func (s *Service) parse(raw string) (CommandSpec, error) {

	var j struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Mode    string   `json:"mode"`
	}

	if json.Unmarshal([]byte(raw), &j) == nil && j.Command != "" {
		mode := normalizeMode(j.Mode)
		if mode == "shell" {
			if strings.EqualFold(j.Command, "sh") && len(j.Args) >= 2 && j.Args[0] == "-c" {
				return CommandSpec{Command: j.Args[1], Mode: "shell"}, nil
			}
			if len(j.Args) == 0 {
				return CommandSpec{Command: j.Command, Mode: "shell"}, nil
			}
			return CommandSpec{}, errors.New("invalid shell command payload: use mode=shell with script or sh -c <script>")
		}
		return normalizeExecSpec(CommandSpec{Command: j.Command, Args: j.Args, Mode: mode}), nil
	}

	args, err := shlex.Split(raw)
	if err != nil {
		return CommandSpec{}, err
	}

	if len(args) == 0 {
		return CommandSpec{}, errors.New("empty command")
	}

	return CommandSpec{
		Command: args[0],
		Args:    args[1:],
		Mode:    "exec",
	}, nil
}

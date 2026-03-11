package command

import (
	"encoding/json"
	"errors"
	"strings"
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
	cmd *Command
}

// NewService returns a ready-to-use command service.
func NewService() *Service {
	return &Service{cmd: &Command{}}
}

// ExecuteFromLLM takes the raw string passed from the model and executes
// the corresponding command.  It understands two representations:
//
//   - a plain shell-style string such as "ls -la /tmp"
//   - a JSON object like
//     {"command":"ls","args":["-la","/tmp"]}
//
// The latter form is useful when the agent reasoning produces structured
// output.  After parsing, the command name is checked against the
// blacklist, and the underlying Command.Exec method is invoked.
func (s *Service) ExecuteFromLLM(raw string) (string, error) {
	cmdName, args, err := s.parse(raw)
	if err != nil {
		return "", err
	}
	if IsBlocked(cmdName) {
		return "", errors.New("command blocked")
	}
	return s.cmd.Exec(cmdName, args)
}

// parse decodes the raw input into a command name and argument slice.
func (s *Service) parse(raw string) (string, []string, error) {
	// try JSON first
	var j struct {
		Command string   `json:"command,omitempty"`
		Args    []string `json:"args,omitempty"`
	}
	if err := json.Unmarshal([]byte(raw), &j); err == nil && j.Command != "" {
		return j.Command, j.Args, nil
	}

	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return "", nil, errors.New("no command provided")
	}
	return fields[0], fields[1:], nil
}

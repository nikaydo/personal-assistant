package command

import (
	"encoding/json"
	"errors"
	"slices"

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
	cmd *Command
}

// type if true then its allowed list if false its blocklist
type CommandList struct {
	Type       bool
	AskAllowed bool
	List       map[string][]string
}

func CheckCommand(cmdToExec string, args []string, c CommandList) bool {
	switch c.Type {
	//allowed to execute
	//if command in List then command allowed to execute
	case true:
		allowedArgs, ok := c.List[cmdToExec]
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
		_, ok := c.List[cmdToExec]
		if !ok {
			return true
		}
		return false
	default:
		return false
	}
}

// NewService returns a ready-to-use command service.
func NewService() *Service {
	return &Service{cmd: &Command{}}
}

func (s *Service) ExecuteFromLLM(raw string, cList CommandList) struct {
	Stdout string `json:"stdout"`
	Stdin  string `json:"stdin"`
	Code   int    `json:"exit_code"`
	Error  string `json:"error"`
} {
	var data struct {
		Stdout string `json:"stdout"`
		Stdin  string `json:"stdin"`
		Code   int    `json:"exit_code"`
		Error  string `json:"error"`
	}
	cmdName, args, err := s.parse(raw)
	if err != nil {
		data.Error = err.Error()
		return data
	}
	if !CheckCommand(cmdName, args, cList) {
		data.Error = "Command not allowed"
		return data
	}

	data.Stdout, data.Stdin, data.Code, err = s.cmd.Exec(cmdName, args...)
	if err != nil {
		data.Error = err.Error()
	}
	data.Error = ""
	return data

}

// parse decodes the raw input into a command name and argument slice.
func (s *Service) parse(raw string) (string, []string, error) {

	var j struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}

	if json.Unmarshal([]byte(raw), &j) == nil && j.Command != "" {
		return j.Command, j.Args, nil
	}

	args, err := shlex.Split(raw)
	if err != nil {
		return "", nil, err
	}

	if len(args) == 0 {
		return "", nil, errors.New("empty command")
	}

	return args[0], args[1:], nil
}

package command

import (
	"bytes"
	"os/exec"
	"strings"
)

type Command struct {
	Allowed []string
}

func (c *Command) Exec(command string, args ...string) (string, string, int, error) {
	s := []string{command}
	s = append(s, args...)

	cmdStr := strings.Join(s, " ")

	cmd := exec.Command("bash", "-c", cmdStr)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", "", -1, err
		}
	}

	return stdout.String(), stderr.String(), exitCode, nil
}

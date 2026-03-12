package command

import (
	"bytes"
	"os/exec"
)

type Command struct {
	Allowed []string
}

func (c *Command) Exec(command string, args ...string) (string, string, int, error) {
	cmd := exec.Command(command, args...)

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

func (c *Command) ExecShell(script string) (string, string, int, error) {
	return c.Exec("sh", "-c", script)
}

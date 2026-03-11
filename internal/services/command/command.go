package command

import "os/exec"

type Command struct {
	Allowed []string
}

func (c *Command) Exec(command string, args []string) (string, error) {
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), err
}

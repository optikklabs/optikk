package hostexec

import (
	"os"
	"os/exec"
	"strconv"
)

// run executes a host command, streaming its output to the CLI's stdio.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func itoa(n int) string { return strconv.Itoa(n) }

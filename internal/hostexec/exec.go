package hostexec

import (
	"os"
	"os/exec"
)

// run executes a host command, streaming its output to the CLI's stdio.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// StartContainer ensures a Podman container is running.
func StartContainer(name string) error {
	return run("podman", "start", name)
}

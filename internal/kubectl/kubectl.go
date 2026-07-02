// Package kubectl runs the kubectl CLI against a named kubeconfig context.
package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Kube addresses one cluster via its kubeconfig context.
type Kube struct {
	Context string
}

// Run executes kubectl and returns stdout; stderr is folded into the error.
func (k Kube) Run(ctx context.Context, args ...string) (string, error) {
	full := append([]string{"--context", k.Context}, args...)
	cmd := exec.CommandContext(ctx, "kubectl", full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("kubectl %s: %w (%s)",
			strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

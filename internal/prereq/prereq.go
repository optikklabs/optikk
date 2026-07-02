// Package prereq verifies the external tools the CLI shells out to.
package prereq

import (
	"fmt"
	"os/exec"
	"strings"
)

// Tool is an external binary the CLI requires on PATH.
type Tool struct {
	Name    string
	Install string
	Docs    string
}

var (
	Podman  = Tool{"podman", "brew install podman", "https://podman.io/docs/installation"}
	Kind    = Tool{"kind", "brew install kind", "https://kind.sigs.k8s.io/docs/user/quick-start/#installation"}
	Kubectl = Tool{"kubectl", "brew install kubectl", "https://kubernetes.io/docs/tasks/tools/"}
)

// Check fails fast with install instructions for every missing tool.
func Check(tools ...Tool) error {
	var missing []string
	for _, t := range tools {
		if _, err := exec.LookPath(t.Name); err != nil {
			missing = append(missing, fmt.Sprintf("  %s — install: %s (%s)", t.Name, t.Install, t.Docs))
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing required tools:\n%s", strings.Join(missing, "\n"))
}

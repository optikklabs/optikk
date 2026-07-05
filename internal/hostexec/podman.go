// Package hostexec handles host-level Podman machine steps that have no Go SDK.
package hostexec

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Floor is the minimum Podman machine sizing required to run the local stack.
// 8 GiB is a hard floor — less and ClickHouse OOMs on boot.
const (
	minCPUs     = 5
	minMemoryMB = 8192
	minDiskGB   = 40
)

// machine is the subset of `podman machine inspect` the CLI reads.
type machine struct {
	Name      string `json:"Name"`
	State     string `json:"State"`
	Rootful   bool   `json:"Rootful"`
	Resources struct {
		CPUs     int `json:"CPUs"`
		DiskSize int `json:"DiskSize"` // GB
		Memory   int `json:"Memory"`   // MB
	} `json:"Resources"`
}

// PrecheckPodman verifies the default machine is running, rootful, and meets
// the sizing floor, returning an actionable error naming the exact command to
// run. It only inspects; the operator applies the fix.
func PrecheckPodman() error {
	m, err := inspect()
	if err != nil {
		return err
	}

	if problem := sizingProblem(m); problem != "" {
		return fmt.Errorf("podman machine %s: %s\n  fix: podman machine stop && podman machine set --cpus %d --memory %d --disk-size %d && podman machine start",
			m.Name, problem, max(m.Resources.CPUs, minCPUs), max(m.Resources.Memory, minMemoryMB), max(m.Resources.DiskSize, minDiskGB))
	}

	if !m.Rootful {
		return fmt.Errorf("podman machine %s is not rootful (kind needs rootful)\n  fix: podman machine stop && podman machine set --rootful && podman machine start", m.Name)
	}

	if !strings.EqualFold(m.State, "running") {
		return fmt.Errorf("podman machine %s is %s\n  fix: podman machine start", m.Name, m.State)
	}
	return nil
}

// SetPidsLimit lifts the kind node container's pids limit so ClickHouse can
// grab its >512 boot threads from the shared Podman budget.
func SetPidsLimit(nodeContainer string) error {
	return run("podman", "update", "--pids-limit=-1", nodeContainer)
}

func sizingProblem(m machine) string {
	var short []string
	if m.Resources.CPUs < minCPUs {
		short = append(short, fmt.Sprintf("%d/%d vCPU", m.Resources.CPUs, minCPUs))
	}
	if m.Resources.Memory < minMemoryMB {
		short = append(short, fmt.Sprintf("%d/%d MB RAM", m.Resources.Memory, minMemoryMB))
	}
	if m.Resources.DiskSize < minDiskGB {
		short = append(short, fmt.Sprintf("%d/%d GB disk", m.Resources.DiskSize, minDiskGB))
	}
	if len(short) == 0 {
		return ""
	}
	return "below floor (" + strings.Join(short, ", ") + ")"
}

func inspect() (machine, error) {
	out, err := exec.Command("podman", "machine", "inspect").Output()
	if err != nil {
		return machine{}, fmt.Errorf("podman machine inspect failed (is Podman installed and initialized?): %w", err)
	}
	var machines []machine
	if err := json.Unmarshal(out, &machines); err != nil {
		return machine{}, fmt.Errorf("parse podman machine inspect: %w", err)
	}
	if len(machines) == 0 {
		return machine{}, fmt.Errorf("no podman machine found; run: podman machine init")
	}
	return machines[0], nil
}

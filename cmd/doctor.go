package cmd

import (
	"fmt"
	"os/exec"

	"github.com/optikklabs/optikk/internal/prereq"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "doctor",
		Aliases:     []string{"check", "preflight"},
		Short:       "Check local prerequisites before provisioning",
		Example:     "  optikk doctor",
		Annotations: map[string]string{annotationNoConfig: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			tools := []prereq.Tool{prereq.Podman, prereq.Kind, prereq.Kubectl}
			if err := prereq.Check(tools...); err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			for _, tool := range tools {
				path, _ := exec.LookPath(tool.Name)
				fmt.Fprintf(w, "ok   %-7s %s\n", tool.Name, path)
			}
			fmt.Fprintln(w, "ok   optikk can provision the local stack")
			return nil
		},
	}
}

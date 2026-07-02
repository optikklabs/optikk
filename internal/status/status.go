// Package status reports pod readiness for the optikk namespace.
package status

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/optikklabs/optikk/internal/kubectl"
)

// Print writes kubectl's pod readiness table for the namespace.
func Print(ctx context.Context, k kubectl.Kube, namespace string, w io.Writer) error {
	out, err := k.Run(ctx, "get", "pods", "-n", namespace)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) == "" {
		fmt.Fprintf(w, "no pods in namespace %q\n", namespace)
		return nil
	}
	_, err = fmt.Fprint(w, out)
	return err
}

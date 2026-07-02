package k8sapply

import (
	"context"
	"strings"

	"github.com/optikklabs/optikk/internal/kubectl"
)

const metricsServerURL = "https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml"

// InstallMetricsServer applies the upstream metrics-server manifest and adds
// the --kubelet-insecure-tls arg (kind's kubelet serving cert is self-signed).
func InstallMetricsServer(ctx context.Context, k kubectl.Kube) error {
	if _, err := k.Run(ctx, "apply", "-f", metricsServerURL,
		"--server-side", "--field-manager", FieldManager, "--force-conflicts"); err != nil {
		return err
	}

	args, err := k.Run(ctx, "get", "deployment", "metrics-server", "-n", "kube-system",
		"-o", "jsonpath={.spec.template.spec.containers[0].args}")
	if err != nil {
		return err
	}
	if strings.Contains(args, "--kubelet-insecure-tls") {
		return nil
	}
	patch := `[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]`
	_, err = k.Run(ctx, "patch", "deployment", "metrics-server", "-n", "kube-system",
		"--type=json", "-p", patch)
	return err
}

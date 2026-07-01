package verify

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// execCHCount runs a COUNT query in the clickhouse pod and returns the number.
func execCHCount(ctx context.Context, cfg *rest.Config, namespace, query string) (int64, error) {
	out, err := execCH(ctx, cfg, namespace, query)
	if err != nil {
		return 0, err
	}
	var n int64
	if _, err := fmt.Sscan(strings.TrimSpace(out), &n); err != nil {
		return 0, fmt.Errorf("parse clickhouse count %q: %w", out, err)
	}
	return n, nil
}

// execCH execs clickhouse-client in clickhouse-0, authenticating with the
// in-pod CLICKHOUSE_PASSWORD env, and returns stdout.
func execCH(ctx context.Context, cfg *rest.Config, namespace, query string) (string, error) {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", err
	}
	shell := fmt.Sprintf(`clickhouse-client --password "$CLICKHOUSE_PASSWORD" --query %q`, query)
	req := cs.CoreV1().RESTClient().Post().
		Resource("pods").Name("clickhouse-0").Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "clickhouse",
			Command:   []string{"sh", "-c", shell},
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return "", err
	}
	var stdout, stderr bytes.Buffer
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr}); err != nil {
		return "", fmt.Errorf("clickhouse exec: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

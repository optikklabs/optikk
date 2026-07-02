package verify

import (
	"context"
	"fmt"
	"strings"

	"github.com/optikklabs/optikk/internal/kubectl"
)

// execCHCount runs a COUNT query in the clickhouse pod and returns the number.
func execCHCount(ctx context.Context, k kubectl.Kube, namespace, query string) (int64, error) {
	out, err := execCH(ctx, k, namespace, query)
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
func execCH(ctx context.Context, k kubectl.Kube, namespace, query string) (string, error) {
	shell := fmt.Sprintf(`clickhouse-client --password "$CLICKHOUSE_PASSWORD" --query %q`, query)
	out, err := k.Run(ctx, "exec", "-n", namespace, "clickhouse-0", "-c", "clickhouse",
		"--", "sh", "-c", shell)
	if err != nil {
		return "", fmt.Errorf("clickhouse exec: %w", err)
	}
	return out, nil
}

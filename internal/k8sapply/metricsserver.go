package k8sapply

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const metricsServerURL = "https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml"

// InstallMetricsServer applies the upstream metrics-server manifest with the
// --kubelet-insecure-tls arg (kind's kubelet serving cert is self-signed).
func InstallMetricsServer(ctx context.Context, applier *Applier) error {
	manifest, err := fetch(ctx, metricsServerURL)
	if err != nil {
		return fmt.Errorf("fetch metrics-server manifest: %w", err)
	}
	objs, err := decode(manifest)
	if err != nil {
		return err
	}
	for _, o := range objs {
		if o.GetKind() == "Deployment" && o.GetName() == "metrics-server" {
			if err := addInsecureTLSArg(o); err != nil {
				return err
			}
		}
	}
	return applier.Apply(ctx, objs)
}

// addInsecureTLSArg appends --kubelet-insecure-tls to the first container.
func addInsecureTLSArg(d *unstructured.Unstructured) error {
	containers, found, err := unstructured.NestedSlice(d.Object, "spec", "template", "spec", "containers")
	if err != nil || !found || len(containers) == 0 {
		return fmt.Errorf("metrics-server deployment has no containers")
	}
	c := containers[0].(map[string]interface{})
	args, _, _ := unstructured.NestedStringSlice(c, "args")
	for _, a := range args {
		if a == "--kubelet-insecure-tls" {
			return nil // already present
		}
	}
	args = append(args, "--kubelet-insecure-tls")
	if err := unstructured.SetNestedStringSlice(c, args, "args"); err != nil {
		return err
	}
	containers[0] = c
	return unstructured.SetNestedSlice(d.Object, containers, "spec", "template", "spec", "containers")
}

func fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

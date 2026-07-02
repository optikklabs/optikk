package k8sapply

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/optikklabs/optikk/internal/kubectl"
)

// workload is the slice of Deployment/StatefulSet JSON the wait logic reads.
type workload struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name       string `json:"name"`
		Generation int64  `json:"generation"`
	} `json:"metadata"`
	Spec struct {
		Replicas *int32 `json:"replicas"`
	} `json:"spec"`
	Status struct {
		ObservedGeneration int64 `json:"observedGeneration"`
		AvailableReplicas  int32 `json:"availableReplicas"`
		ReadyReplicas      int32 `json:"readyReplicas"`
	} `json:"status"`
}

// WaitRollouts blocks until every Deployment and StatefulSet in the namespace
// reports its desired replicas ready, or the timeout elapses.
func WaitRollouts(ctx context.Context, k kubectl.Kube, namespace string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		notReady, err := unreadyWorkloads(ctx, k, namespace)
		if err != nil {
			return err
		}
		if len(notReady) == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s", timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// unreadyWorkloads returns names of workloads not yet at desired readiness.
func unreadyWorkloads(ctx context.Context, k kubectl.Kube, namespace string) ([]string, error) {
	out, err := k.Run(ctx, "get", "deployments,statefulsets", "-n", namespace, "-o", "json")
	if err != nil {
		return nil, err
	}
	var list struct {
		Items []workload `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, fmt.Errorf("parse workloads: %w", err)
	}

	var pending []string
	for _, w := range list.Items {
		desired := int32(1)
		if w.Spec.Replicas != nil {
			desired = *w.Spec.Replicas
		}
		stale := w.Status.ObservedGeneration < w.Metadata.Generation
		switch w.Kind {
		case "Deployment":
			if stale || w.Status.AvailableReplicas < desired {
				pending = append(pending, "deploy/"+w.Metadata.Name)
			}
		case "StatefulSet":
			if stale || w.Status.ReadyReplicas < desired {
				pending = append(pending, "statefulset/"+w.Metadata.Name)
			}
		}
	}
	return pending, nil
}

// PendingSummary reports which workloads are not ready, for error messages.
func PendingSummary(ctx context.Context, k kubectl.Kube, namespace string) string {
	pending, err := unreadyWorkloads(ctx, k, namespace)
	if err != nil {
		return err.Error()
	}
	if len(pending) == 0 {
		return "all workloads ready"
	}
	return fmt.Sprintf("not ready: %s", strings.Join(pending, ", "))
}

package k8sapply

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// WaitRollouts blocks until every Deployment and StatefulSet in the namespace
// reports its desired replicas ready, or the timeout elapses.
func WaitRollouts(ctx context.Context, cfg *rest.Config, namespace string, timeout time.Duration) error {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	return wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		notReady, err := unreadyWorkloads(ctx, cs, namespace)
		if err != nil {
			return false, err
		}
		if len(notReady) == 0 {
			return true, nil
		}
		return false, nil
	})
}

// unreadyWorkloads returns names of workloads not yet at desired readiness.
func unreadyWorkloads(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]string, error) {
	var pending []string

	deploys, err := cs.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, d := range deploys.Items {
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		if d.Status.ObservedGeneration < d.Generation || d.Status.AvailableReplicas < desired {
			pending = append(pending, "deploy/"+d.Name)
		}
	}

	sts, err := cs.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, s := range sts.Items {
		desired := int32(1)
		if s.Spec.Replicas != nil {
			desired = *s.Spec.Replicas
		}
		if s.Status.ObservedGeneration < s.Generation || s.Status.ReadyReplicas < desired {
			pending = append(pending, "statefulset/"+s.Name)
		}
	}
	return pending, nil
}

// PendingSummary reports which workloads are not ready, for error messages.
func PendingSummary(ctx context.Context, cfg *rest.Config, namespace string) string {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err.Error()
	}
	pending, err := unreadyWorkloads(ctx, cs, namespace)
	if err != nil {
		return err.Error()
	}
	if len(pending) == 0 {
		return "all workloads ready"
	}
	return fmt.Sprintf("not ready: %s", strings.Join(pending, ", "))
}

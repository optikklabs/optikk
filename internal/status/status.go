// Package status reports pod readiness for the optikk namespace.
package status

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Print writes a pod readiness table for the namespace.
func Print(ctx context.Context, cfg *rest.Config, namespace string, w io.Writer) error {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	pods, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		fmt.Fprintf(w, "no pods in namespace %q\n", namespace)
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE")
	for _, p := range pods.Items {
		ready, total := readiness(p)
		fmt.Fprintf(tw, "%s\t%d/%d\t%s\t%d\t%s\n",
			p.Name, ready, total, p.Status.Phase, restarts(p), age(p))
	}
	return tw.Flush()
}

func readiness(p corev1.Pod) (ready, total int) {
	for _, c := range p.Status.ContainerStatuses {
		total++
		if c.Ready {
			ready++
		}
	}
	return ready, total
}

func restarts(p corev1.Pod) int32 {
	var n int32
	for _, c := range p.Status.ContainerStatuses {
		n += c.RestartCount
	}
	return n
}

func age(p corev1.Pod) string {
	d := time.Since(p.CreationTimestamp.Time).Round(time.Second)
	return d.String()
}

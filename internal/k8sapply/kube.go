package k8sapply

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// EnsureNamespace creates the namespace if it does not already exist.
func EnsureNamespace(ctx context.Context, cfg *rest.Config, name string) error {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	_, err = cs.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// EnsureSecret creates or updates an opaque secret with string data.
func EnsureSecret(ctx context.Context, cfg *rest.Config, namespace, name string, data map[string]string) error {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		StringData: data,
	}
	_, err = cs.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		_, err = cs.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	}
	return err
}

// TraefikLoadBalancerIP returns the external IP of the traefik Service, or ""
// if not yet assigned.
func TraefikLoadBalancerIP(ctx context.Context, cfg *rest.Config, namespace string) (string, error) {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", err
	}
	svc, err := cs.CoreV1().Services(namespace).Get(ctx, "traefik", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	for _, ing := range svc.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			return ing.IP, nil
		}
		if ing.Hostname != "" {
			return ing.Hostname, nil
		}
	}
	return "", nil
}

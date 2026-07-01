// Package adminseed sets the query super-admin credentials and reseeds.
package adminseed

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// SetCredentials writes the admin email/password into query-secret and
// restarts query so EnsureSuperAdmin runs. Note: query seeds create-if-absent,
// so this only creates a new admin when none exists yet.
func SetCredentials(ctx context.Context, cfg *rest.Config, namespace, email, password string) error {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	secret, err := cs.CoreV1().Secrets(namespace).Get(ctx, "query-secret", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get query-secret: %w", err)
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	secret.Data["admin-email"] = []byte(email)
	secret.Data["admin-password"] = []byte(password)
	if _, err := cs.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update query-secret: %w", err)
	}

	return restartDeployment(ctx, cs, namespace, "query")
}

// restartDeployment triggers a rollout by stamping the restart annotation.
func restartDeployment(ctx context.Context, cs *kubernetes.Clientset, namespace, name string) error {
	patch := fmt.Sprintf(
		`{"spec":{"template":{"metadata":{"annotations":{"optikk.dev/restartedAt":%q}}}}}`,
		time.Now().Format(time.RFC3339))
	_, err := cs.AppsV1().Deployments(namespace).Patch(
		ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("restart %s: %w", name, err)
	}
	return nil
}

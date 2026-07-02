// Package adminseed sets the query super-admin credentials and reseeds.
package adminseed

import (
	"context"
	"fmt"

	"github.com/optikklabs/optikk/internal/kubectl"
)

// SetCredentials writes the admin email/password into query-secret and
// restarts query so EnsureSuperAdmin runs. Note: query seeds create-if-absent,
// so this only creates a new admin when none exists yet.
func SetCredentials(ctx context.Context, k kubectl.Kube, namespace, email, password string) error {
	patch := fmt.Sprintf(`{"stringData":{"admin-email":%q,"admin-password":%q}}`, email, password)
	if _, err := k.Run(ctx, "patch", "secret", "query-secret", "-n", namespace, "-p", patch); err != nil {
		return fmt.Errorf("update query-secret: %w", err)
	}
	if _, err := k.Run(ctx, "rollout", "restart", "deployment/query", "-n", namespace); err != nil {
		return fmt.Errorf("restart query: %w", err)
	}
	return nil
}

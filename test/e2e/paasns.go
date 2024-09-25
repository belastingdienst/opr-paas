package e2e

import (
	"context"
	"fmt"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"

	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// waitForPaasNSReconciliation polls a PaasNS resource, blocking until the status reports successful reconciliation.
func waitForPaasNSReconciliation(ctx context.Context, cfg *envconf.Config, paasns *api.PaasNS) error {
	waitCond := conditions.New(cfg.Client().Resources()).
		ResourceMatch(paasns, func(object k8s.Object) bool {
			messages := object.(*api.PaasNS).Status.Messages

			return reconcileStatusRegexp.MatchString(messages[len(messages)-1])
		})

	if err := waitForDefaultOpts(ctx, waitCond); err != nil {
		return fmt.Errorf("failed waiting for PaasNS %s to be reconciled: %w", paasns.GetName(), err)
	}

	return nil
}

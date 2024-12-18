package e2e

import (
	"context"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// waitForPaasNSReconciliation polls a PaasNs resource, blocking until the Conditions report successful reconciliation.
func waitForPaasNSReconciliation(ctx context.Context, cfg *envconf.Config, paasns *api.PaasNS, oldGeneration int64) error {
	return waitForCondition(ctx, cfg, paasns, oldGeneration, api.TypeReadyPaasNs)
}

package e2e

import (
	"context"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// waitForPaasNSReconciliation polls a PaasNs resource, blocking until the Conditions report successful reconciliation.
func waitForPaasNSReconciliation(ctx context.Context, cfg *envconf.Config, paasns *api.PaasNS, oldGeneration int64) error {
	return waitForStatus(ctx, cfg, paasns, oldGeneration, func(conds []metav1.Condition) bool {
		return meta.IsStatusConditionTrue(conds, api.TypeReadyPaasNs)
	})
}

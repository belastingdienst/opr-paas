package e2e

import (
	"context"
	"fmt"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// waitForPaasNSReconciliation polls a PaasNs resource, blocking until the Conditions report successful reconciliation.
func waitForPaasNSReconciliation(ctx context.Context, cfg *envconf.Config, paasns *api.PaasNS, oldGeneration int64) error {
	waitCond := conditions.New(cfg.Client().Resources()).
		ResourceMatch(paasns, func(object k8s.Object) bool {
			conditionsMet := meta.IsStatusConditionPresentAndEqual(object.(*api.PaasNS).Status.Conditions, api.TypeReadyPaasNs, metav1.ConditionTrue)
			if conditionsMet {
				foundCondition := meta.FindStatusCondition(object.(*api.PaasNS).Status.Conditions, api.TypeReadyPaasNs)
				versionMet := object.(*api.PaasNS).Generation != oldGeneration && object.(*api.PaasNS).Generation == foundCondition.ObservedGeneration
				return conditionsMet && versionMet
			}
			return false
		})

	if err := waitForDefaultOpts(ctx, waitCond); err != nil {
		err = cfg.Client().Resources().Get(ctx, paasns.GetName(), paasns.Namespace, paasns)
		if err != nil {
			return fmt.Errorf("could not get paasns resource which is waited for: %w", err)
		}
		return fmt.Errorf("failed waiting for PaasNS %s to be reconciled: %w and has status block: %v", paasns.GetName(), err, paasns.Status)
	}

	return nil
}

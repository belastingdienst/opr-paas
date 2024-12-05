package e2e

// Helper functions for manipulating Paas resources in a test.

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// getPaas retrieves the Paas with the associated name.
func getPaas(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) *api.Paas {
	return getOrFail(ctx, name, cfg.Namespace(), &api.Paas{}, t, cfg)
}

// createPaasSync requests Paas creation and returns once it has reconciled.
func createPaasSync(ctx context.Context, cfg *envconf.Config, paas *api.Paas) error {
	if err := cfg.Client().Resources().Create(ctx, paas); err != nil {
		return fmt.Errorf("failed to create Paas %s: %w", paas.GetName(), err)
	}

	return waitForPaasReconciliation(ctx, cfg, paas, 0)
}

// updatePaasSync requests an update to a Paas and returns once the Paas reports successful reconciliation.
func updatePaasSync(ctx context.Context, cfg *envconf.Config, paas *api.Paas) error {
	oldGeneration := paas.Generation
	if err := cfg.Client().Resources().Update(ctx, paas); err != nil {
		return fmt.Errorf("failed to update Paas %s: %w", paas.GetName(), err)
	}

	return waitForPaasReconciliation(ctx, cfg, paas, oldGeneration)
}

// deletePaasSync deletes the Paas with the associated name.
func deletePaasSync(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) {
	paas := &api.Paas{ObjectMeta: metav1.ObjectMeta{Name: name}}

	if err := deleteResourceSync(ctx, cfg, paas); err != nil {
		t.Fatal(err)
	}
}

// waitForPaasReconciliation polls a Paas resource, blocking until the Paas status reports successful reconciliation.
func waitForPaasReconciliation(ctx context.Context, cfg *envconf.Config, paas *api.Paas, oldGeneration int64) error {
	waitCond := conditions.New(cfg.Client().Resources()).
		ResourceMatch(paas, func(object k8s.Object) bool {
			conditionsMet := meta.IsStatusConditionPresentAndEqual(object.(*api.Paas).Status.Conditions, api.TypeReadyPaas, metav1.ConditionTrue)
			if conditionsMet {
				foundCondition := meta.FindStatusCondition(object.(*api.Paas).Status.Conditions, api.TypeReadyPaas)
				versionMet := object.(*api.Paas).Generation != oldGeneration && object.(*api.Paas).Generation == foundCondition.ObservedGeneration
				return conditionsMet && versionMet
			}
			return false
		})

	if err := waitForDefaultOpts(ctx, waitCond); err != nil {
		err = cfg.Client().Resources().Get(ctx, paas.GetName(), paas.Namespace, paas)
		if err != nil {
			return fmt.Errorf("could not get paas resource which was waited for: %w", err)
		}
		return fmt.Errorf("failed waiting for Paas %s to be reconciled: %w and has status block: %v", paas.GetName(), err, paas.Status)
	}

	return nil
}

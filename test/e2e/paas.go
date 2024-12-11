package e2e

// Helper functions for manipulating Paas resources in a test.

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// getPaas retrieves the Paas with the associated name.
func getPaas(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) *api.Paas {
	return getOrFail(ctx, name, cfg.Namespace(), &api.Paas{}, t, cfg)
}

// createPaasSync requests Paas creation and returns once it has reconciled.
func createPaasSync(ctx context.Context, cfg *envconf.Config, paas *api.Paas) error {
	return createSyncStatus(ctx, cfg, paas, func(conds []metav1.Condition) bool {
		return meta.IsStatusConditionTrue(conds, api.TypeReadyPaas)
	})
}

// updatePaasSync requests an update to a Paas and returns once the Paas reports successful reconciliation.
func updatePaasSync(ctx context.Context, cfg *envconf.Config, paas *api.Paas) error {
	return updateSyncStatus(ctx, cfg, paas, func(conds []metav1.Condition) bool {
		return meta.IsStatusConditionTrue(conds, api.TypeReadyPaas)
	})
}

// deletePaasSync deletes the Paas with the associated name.
func deletePaasSync(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) {
	paas := &api.Paas{ObjectMeta: metav1.ObjectMeta{Name: name}}

	if err := deleteResourceSync(ctx, cfg, paas); err != nil {
		t.Fatal(err)
	}
}

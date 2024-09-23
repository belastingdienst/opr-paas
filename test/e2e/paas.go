package e2e

// Helper functions for manipulating Paas resources in a test.

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var reconcileStatusRegexp = regexp.MustCompile("^INFO: reconcile for .* succeeded$")

// getPaas retrieves the Paas with the associated name.
func getPaas(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) *api.Paas {
	return getOrFail(ctx, name, cfg.Namespace(), &api.Paas{}, t, cfg)
}

// createPaasSync requests Paas creation and returns once it has reconciled.
func createPaasSync(ctx context.Context, cfg *envconf.Config, paas *api.Paas) error {
	if err := cfg.Client().Resources().Create(ctx, paas); err != nil {
		return fmt.Errorf("failed to create Paas %s: %w", paas.GetName(), err)
	}

	return waitForPaasReconciliation(ctx, cfg, paas)
}

// updatePaasSync requests an update to a Paas and returns once the Paas reports successful reconciliation.
func updatePaasSync(ctx context.Context, cfg *envconf.Config, paas *api.Paas) error {
	if err := cfg.Client().Resources().Update(ctx, paas); err != nil {
		return fmt.Errorf("failed to update Paas %s: %w", paas.GetName(), err)
	}

	return waitForPaasReconciliation(ctx, cfg, paas)
}

// deletePaasSync deletes the Paas with the associated name.
func deletePaasSync(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) {
	paas := &api.Paas{ObjectMeta: metav1.ObjectMeta{Name: name}}

	if err := deleteResourceSync(ctx, cfg, paas); err != nil {
		t.Fatal(err)
	}
}

// waitForPaasReconciliation polls a Paas resource, blocking until the Paas status reports successful reconciliation.
func waitForPaasReconciliation(ctx context.Context, cfg *envconf.Config, paas *api.Paas) error {
	waitCond := conditions.New(cfg.Client().Resources()).
		ResourceMatch(paas, func(object k8s.Object) bool {
			messages := object.(*api.Paas).Status.Messages

			return reconcileStatusRegexp.MatchString(messages[len(messages)-1])
		})

	if err := waitForDefaultOpts(ctx, waitCond); err != nil {
		return fmt.Errorf("failed waiting for Paas %s to be reconciled: %w", paas.GetName(), err)
	}

	return nil
}

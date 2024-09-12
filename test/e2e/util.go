package e2e

import (
	"context"
	"testing"
	"time"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

// Duration to pause after PaaS creation to wait for reconciliation.
const waitForOperatorDuration = 1 * time.Second

func waitForOperator() {
	time.Sleep(waitForOperatorDuration)
}

// createPaasFn accepts a PaaS spec object and a name and creates the PaaS resource.
func createPaasFn(name string, paasSpec api.PaasSpec) types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		paas := &api.Paas{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       paasSpec,
		}

		if err := cfg.Client().Resources().Create(ctx, paas); err != nil {
			t.Fatalf("Failed to create PaaS resource: %v", err)
		}

		waitForOperator()

		return ctx
	}
}

// teardownPaasFn deletes the PaaS if it still exists (e.g. if deleting the PaaS is not part of the test steps, or if an
// earlier assertion failed causing the deletion step to be skipped).
// Can be called as `.Teardown(teardownPaasFn("paas-name"))`
func teardownPaasFn(paasName string) types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		paas := &api.Paas{ObjectMeta: metav1.ObjectMeta{Name: paasName}}

		if cfg.Client().Resources().Delete(ctx, paas) == nil {
			t.Logf("Paas %s deleted", paasName)
		}

		return ctx
	}
}

// getPaas retrieves the PaaS with the associated name.
func getPaas(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) api.Paas {
	var paas api.Paas

	if err := cfg.Client().Resources().Get(ctx, name, cfg.Namespace(), &paas); err != nil {
		t.Fatalf("Failed to retrieve PaaS: %v", err)
	}

	return paas
}

// deletePaas deletes the PaaS with the associated name.
func deletePaas(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) {
	paas := &api.Paas{ObjectMeta: metav1.ObjectMeta{Name: name}}

	if err := cfg.Client().Resources().Delete(ctx, paas); err != nil {
		t.Fatalf("Failed to delete PaaS: %v", err)
	}

	waitForOperator()
}

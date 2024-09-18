package e2e

// Reusable step functions for end-to-end tests

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

// createPaasFn accepts a Paas spec object and a name and creates the Paas resource.
func createPaasFn(name string, paasSpec api.PaasSpec) types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		paas := &api.Paas{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       paasSpec,
		}

		if err := createPaasSync(ctx, cfg, paas); err != nil {
			t.Fatal(err)
		}

		return ctx
	}
}

// teardownPaasFn deletes the Paas if it still exists (e.g. if deleting the Paas is not part of the test steps, or if an
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

package e2e

// Reusable step functions for end-to-end tests

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/v2/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

// createPaasFn accepts a valid Paas spec object and a name and creates the Paas resource,
// waiting for successful creation.
func createPaasFn(name string, paasSpec api.PaasSpec) types.StepFunc {
	return createPaasWithCondFn(name, paasSpec, api.TypeReadyPaas)
}

// createPaasWithCondFn accepts an invalid Paas spec object and a name and creates the Paas resource,
// waiting for the given condition to be true.
func createPaasWithCondFn(name string, paasSpec api.PaasSpec, readyCondition string) types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		paas := &api.Paas{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       paasSpec,
		}

		if err := createSync(ctx, cfg, paas, readyCondition); err != nil {
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

		// Paas is deleted synchronously to prevent race conditions between test invocations
		_ = deleteResourceSync(ctx, cfg, paas)

		return ctx
	}
}

package e2e

// Helper functions for manipulating Paas resources in a test.

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/v3/api/v1alpha2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// getPaas retrieves the Paas with the associated name.
func getPaas(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) *api.Paas {
	return getOrFail(ctx, name, cfg.Namespace(), &api.Paas{}, t, cfg)
}

// deletePaasSync deletes the Paas with the associated name.
func deletePaasSync(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) {
	paas := &api.Paas{ObjectMeta: metav1.ObjectMeta{Name: name}}

	if err := deleteResourceSync(ctx, cfg, paas); err != nil {
		t.Fatal(err)
	}
}

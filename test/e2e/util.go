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

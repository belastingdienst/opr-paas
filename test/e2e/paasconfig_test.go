package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestPaasConfig(t *testing.T) {
	testenv.Test(
		t,
		features.New("PaasConfig").
			Assess("PaasConfig is Active", assertPaasConfigIsActive).
			Feature(),
	)
}

func assertPaasConfigIsActive(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	var fetchedPaasConfig api.PaasConfig

	// Ensure PaasConfig exists
	err := cfg.Client().Resources().Get(ctx, "paas-config", "", &fetchedPaasConfig)
	require.NoError(t, err)

	expectedStatus := v1.ConditionStatus("True")
	condition := fetchedPaasConfig.Status.Conditions[0]
	assert.Equal(t, expectedStatus, condition.Status)

	expectedStatus = "Active"
	assert.Equal(t, expectedStatus, condition.Type)

	return ctx
}

package e2e

import (
	"context"
	"testing"

	v1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestPaasConfig(t *testing.T) {
	testenv.Test(
		t,
		features.New("PaasConfig is a Singleton").
			Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				return ctx
			}).
			Assess("A single PaasConfig resource instances exists", assertOnePaasConfigExists).
			// Assess("Adding a second PaasConfig fails", assertPaasConfigCannotBeAdded).
			Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				return ctx
			}).
			Feature(),
		// features.New("Paas Operator can run without a PaasConfig Resource").
		// 	Feature(),
		// features.New("PaasConfig is valid").
		// 	Assess("PaasConfig is Active", assertPaasConfigIsActive).
		// 	Assess("PaasConfig is Updated", assertPaasConfigIsUpdated).
		// 	Assess("PaasConfig Invalid Spec", assertPaasConfigInvalidSpec).
		// 	Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		// 		return ctx
		// 	}).
		// 	Feature(),
	)
}

// assertOnePaasConfigExists verifies that only one PaasConfig resource exists.
func assertOnePaasConfigExists(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	// Ensure PaasConfig does not exist
	paasconfiglist := v1alpha1.PaasConfigList{}
	err := cfg.Client().Resources().List(ctx, &paasconfiglist)
	require.NoError(t, err)
	require.NotEmpty(t, paasconfiglist.Items)
	assert.Equal(t, "paas-config", paasconfiglist.Items[0].Name)

	return ctx
}

// assertPaasConfigIsActive verifies that the PaasConfig resource exists and is active.
func assertPaasConfigIsActive(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	var paasConfig v1alpha1.PaasConfig

	// Ensure PaasConfig exists
	err := cfg.Client().Resources().Get(ctx, "paas-config", "", &paasConfig)
	require.NoError(t, err)

	// Ensure we have Active status on PaasConfig
	require.NotEmpty(t, paasConfig.Status.Conditions, "PaasConfig status conditions are empty")
	assert.True(t, meta.IsStatusConditionPresentAndEqual(
		paasConfig.Status.Conditions,
		v1alpha1.TypeActivePaasConfig,
		metav1.ConditionTrue),
		"PaasConfig is not active")

	return ctx
}

func assertPaasConfigIsUpdated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	// Retrieve the existing PaasConfig
	var paasConfig v1alpha1.PaasConfig
	err := cfg.Client().Resources().Get(ctx, "paas-config", "", &paasConfig)
	require.NoError(t, err, "Failed to get PaasConfig")

	// Modify the PaasConfig spec
	originalDebug := paasConfig.Spec.Debug
	paasConfig.Spec.Debug = !originalDebug // Toggle the debug flag

	// Update the PaasConfig
	err = cfg.Client().Resources().Update(ctx, &paasConfig)
	require.NoError(t, err, "Failed to update PaasConfig")

	// Retrieve the updated PaasConfig
	var updatedPaasConfig v1alpha1.PaasConfig
	err = cfg.Client().Resources().Get(ctx, "paas-config", "", &updatedPaasConfig)
	require.NoError(t, err, "Failed to get updated PaasConfig")

	// Verify the changes
	require.Equal(t, !originalDebug, updatedPaasConfig.Spec.Debug, "PaasConfig Debug flag did not update correctly")

	return ctx
}

func assertPaasConfigInvalidSpec(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	// Define an invalid PaasConfig (e.g., missing required fields)
	invalidPaasConfig := &v1alpha1.PaasConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "invalid-paas-config",
		},
		// Spec is intentionally invalid or incomplete
	}

	// Try to create the invalid PaasConfig
	err := cfg.Client().Resources().Create(ctx, invalidPaasConfig)
	require.Error(t, err, "Expected error when creating invalid PaasConfig")

	// Verify that the invalid PaasConfig does not exist
	var paasConfig v1alpha1.PaasConfig
	err = cfg.Client().Resources().Get(ctx, "invalid-paas-config", "", &paasConfig)
	require.Error(t, err, "Expected error when getting invalid PaasConfig")
	require.True(t, apierrors.IsNotFound(err), "Expected NotFound error, got: %v", err)

	return ctx
}

func assertPaasReconciliationAfterConfigUpdate(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paasConfig := examplePaasConfig.DeepCopy()
	require.NoError(t, createSync(ctx, cfg, paasConfig, v1alpha1.TypeActivePaasConfig))

	paas := &v1alpha1.Paas{
		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
		Spec: v1alpha1.PaasSpec{
			Requestor: "paas-user",
			Quota:     make(quota.Quota),
		},
	}
	require.NoError(t, createSync(ctx, cfg, paas, v1alpha1.TypeReadyPaas))

	ns := getOrFail(ctx, "foo", cfg.Namespace(), &v1.Namespace{}, t, cfg)
	assert.Equal(t, "paas-user", ns.Labels["o.lbl"])
	assert.Equal(t, "foo", ns.Labels["q.lbl"])

	paasConfig.Spec.RequestorLabel = "another.lbl"
	paasConfig.Spec.QuotaLabel = "different.lbl"
	require.NoError(t, updateSync(ctx, cfg, paasConfig, v1alpha1.TypeActivePaasConfig))

	ns = getOrFail(ctx, "foo", cfg.Namespace(), &v1.Namespace{}, t, cfg)
	assert.Equal(t, "paas-user", ns.Labels["another.lbl"])
	assert.Equal(t, "foo", ns.Labels["different.lbl"])

	return ctx
}

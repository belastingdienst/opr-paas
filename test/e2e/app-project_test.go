package e2e

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"
	argo "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	simplePaas      = "paasje"
	testAppSet      = "ssoas"
	appSetNamespace = "asns"
)

func TestAppProject(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: "paas-user",
		Quota:     make(quota.Quotas),
		Capabilities: api.PaasCapabilities{
			SSO: api.PaasSSO{Enabled: true},
		},
	}

	testenv.Test(
		t,
		features.New("App Project").
			Setup(createPaasFn(simplePaas, paasSpec)).
			Assess("is created", assertAppProjectCreated).
			Assess("is deleted when PaaS is deleted", assertAppProjectDeleted).
			Teardown(teardownPaasFn(simplePaas)).
			Feature(),
	)
}

func assertAppProjectCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, simplePaas, t, cfg)
	namespace := getOrFail(ctx, simplePaas, cfg.Namespace(), &corev1.Namespace{}, t, cfg)
	applicationSet := getOrFail(ctx, testAppSet, appSetNamespace, &argo.ApplicationSet{}, t, cfg)
	appProject := getOrFail(ctx, simplePaas, appSetNamespace, &argo.AppProject{}, t, cfg)

	// ClusterResource is created with the same name as the PaaS
	assert.Equal(t, simplePaas, paas.Name)

	// Paas Namespace exist
	assert.Equal(t, simplePaas, namespace.Name)

	// SSO should be enabled
	assert.True(t, paas.Spec.Capabilities.SSO.Enabled)

	// ApplicationSet exist
	assert.NotEmpty(t, applicationSet)

	// AppProject exist
	assert.NotEmpty(t, appProject)

	// The owner of the AppProject is the Paas that created it
	assert.Equal(t, paas.UID, appProject.OwnerReferences[0].UID)

	assert.Equal(t, []string{
		"resources-finalizer.argocd.argoproj.io",
	}, appProject.Finalizers)

	applicationSetListEntries, appSetListEntriesError := getApplicationSetListEntries(applicationSet)

	// List entries should not be empty
	require.NoError(t, appSetListEntriesError)
	assert.Len(t, applicationSetListEntries, 1)

	// At least one JSON object should have "paas": "paasnaam"
	assert.Equal(t, simplePaas, applicationSetListEntries[0]["paas"])

	return ctx
}

func assertAppProjectDeleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaasSync(ctx, simplePaas, t, cfg)
	appProject := getOrFail(ctx, simplePaas, appSetNamespace, &argo.AppProject{}, t, cfg)

	// ApplicationSet is deleted
	applicationSet := getOrFail(ctx, testAppSet, appSetNamespace, &argo.ApplicationSet{}, t, cfg)
	applicationSetListEntries, appSetListEntriesError := getApplicationSetListEntries(applicationSet)

	// List Entries should be empty
	require.NoError(t, appSetListEntriesError)
	assert.Empty(t, applicationSetListEntries)

	// AppProject still exists due to finalizer not being removed as there is no active GitOps operator
	assert.NotEmpty(t, appProject)
	assert.NotEmpty(t, appProject.DeletionTimestamp)

	// Mock GitOps operator behavior which will normally remove finalizer
	appProject.SetFinalizers(nil)
	err := cfg.Client().Resources().Update(ctx, appProject)
	require.NoError(t, err)

	// Assert removal of AppProject
	err = cfg.Client().Resources().Get(ctx, simplePaas, appSetNamespace, appProject)
	assert.Error(t, err, errors.IsNotFound)

	return ctx
}

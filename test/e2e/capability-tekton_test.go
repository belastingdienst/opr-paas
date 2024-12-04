package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"
	argo "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"

	quotav1 "github.com/openshift/api/quota/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasWithCapabilityTekton = "tpaas"
	paasTektonNs             = "tpaas-tekton"
	paasTektonCRQ            = "paas-tekton"
	TektonApplicationSet     = "tektonas"
	asTektonNamespace        = "asns"
)

func TestCapabilityTekton(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: "paas-user",
		Quota:     make(quota.Quota),
		Capabilities: api.PaasCapabilities{
			"tekton": api.PaasCapability{
				Enabled: true,
			},
		},
	}

	testenv.Test(
		t,
		features.New("Capability Tekton").
			Setup(createPaasFn(paasWithCapabilityTekton, paasSpec)).
			Assess("is created", assertCapTektonCreated).
			Assess("has ClusterRoleBindings", assertTektonCRB).
			Assess("is deleted when Paas is deleted", assertCapTektonDeleted).
			Teardown(teardownPaasFn(paasWithCapabilityTekton)).
			Feature(),
	)
}

func assertCapTektonCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithCapabilityTekton, t, cfg)
	tpaasns := getOrFail(ctx, "tekton", paasWithCapabilityTekton, &api.PaasNS{}, t, cfg)
	require.NoError(t, waitForPaasNSReconciliation(ctx, cfg, tpaasns), "Tekton PaasNS reconciliation succeeds")

	namespace := getOrFail(ctx, paasWithCapabilityTekton, cfg.Namespace(), &corev1.Namespace{}, t, cfg)
	applicationSet := getOrFail(ctx, TektonApplicationSet, asTektonNamespace, &argo.ApplicationSet{}, t, cfg)
	TektonQuota := getOrFail(ctx, paasTektonCRQ, cfg.Namespace(), &quotav1.ClusterResourceQuota{}, t, cfg)

	// ClusterResource is created with the same name as the Paas
	assert.Equal(t, paasWithCapabilityTekton, paas.Name)

	// Paas Namespace exist
	assert.Equal(t, paasWithCapabilityTekton, namespace.Name)

	// Tekton should be enabled
	assert.True(t, paas.Spec.Capabilities.IsCap("tekton"))

	// ApplicationSet exist
	assert.NotEmpty(t, applicationSet)

	applicationSetListEntries, appSetListEntriesError := getApplicationSetListEntries(applicationSet)

	// List entries should not be empty
	require.NoError(t, appSetListEntriesError)
	assert.Len(t, applicationSetListEntries, 1)

	// At least one JSON object should have "paas": "paasnaam"
	assert.Equal(t, paasWithCapabilityTekton, applicationSetListEntries[0]["paas"])

	// Check whether the LabelSelector is specific to the paasnaam-Tekton namespace
	labelSelector := TektonQuota.Spec.Selector.LabelSelector
	assert.True(t, MatchLabelExists(labelSelector.MatchLabels, "q.lbl", paasTektonCRQ))
	assert.False(t, MatchLabelExists(labelSelector.MatchLabels, "q.lbl", "wrong-value"))
	assert.False(t, MatchLabelExists(labelSelector.MatchLabels, "nonexistent.lbl", paasTektonNs))

	// Quota namespace name
	assert.Equal(t, paasTektonCRQ, TektonQuota.Name)

	return ctx
}

func assertCapTektonDeleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaasSync(ctx, paasWithCapabilityTekton, t, cfg)

	// Quota is deleted
	var quotaList quotav1.ClusterResourceQuotaList
	if err := cfg.Client().Resources().List(ctx, &quotaList); err != nil {
		t.Fatalf("Failed to retrieve Quota list: %v", err)
	}

	// Quota list not contains paas
	assert.NotContains(t, quotaList.Items, paasTektonNs)

	// Namespace is deleted
	var namespaceList corev1.NamespaceList
	if err := cfg.Client().Resources().List(ctx, &namespaceList); err != nil {
		t.Fatalf("Failed to retrieve Namespace list: %v", err)
	}

	// Namespace list not contains paas
	assert.NotContains(t, namespaceList.Items, paasWithCapabilityTekton)

	// ApplicationSet is deleted
	applicationSet := getOrFail(ctx, TektonApplicationSet, asTektonNamespace, &argo.ApplicationSet{}, t, cfg)
	applicationSetListEntries, appSetListEntriesError := getApplicationSetListEntries(applicationSet)

	// List Entries should be empty
	require.NoError(t, appSetListEntriesError)
	assert.Empty(t, applicationSetListEntries)

	return ctx
}

func assertTektonCRB(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	for _, crbName := range []string{"paas-view", "paas-alert-routing-edit"} {
		argo_role_binding := getOrFail(ctx, crbName, "", &rbac.ClusterRoleBinding{}, t, cfg)
		subjects := argo_role_binding.Subjects
		assert.Len(t, subjects, 1, "ClusterRoleBinding %s contains one subject", crbName)
		subject := subjects[0]
		assert.Equal(t, "ServiceAccount", subject.Kind, "Subject is of type ServiceAccount")
		assert.Equal(t, paasTektonNs, subject.Namespace, "Subject is of type ServiceAccount")
		assert.Equal(t, "pipeline", subject.Name, "Subject name is tekton")
	}
	return ctx
}

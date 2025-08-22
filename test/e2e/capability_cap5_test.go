package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	argo "github.com/belastingdienst/opr-paas/v3/internal/stubs/argoproj/v1alpha1"
	"github.com/belastingdienst/opr-paas/v3/pkg/quota"

	quotav1 "github.com/openshift/api/quota/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasWithCapability5 = "cap5paas"
	paasCap5Ns          = "cap5paas-cap5"
	cap5ApplicationSet  = "cap5as"
)

func TestCapabilityCap5(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: "paas-user",
		Quota:     make(quota.Quota),
		Capabilities: api.PaasCapabilities{
			"cap5": api.PaasCapability{},
		},
	}

	testenv.Test(
		t,
		features.New("Capability 5").
			Setup(createPaasFn(paasWithCapability5, paasSpec)).
			Assess("is created", assertCap5Created).
			Assess("is deleted when PaaS is deleted", assertCap5Deleted).
			Teardown(teardownPaasFn(paasWithCapability5)).
			Feature(),
	)
}

func assertCap5Created(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithCapability5, t, cfg)
	namespace := getOrFail(ctx, paasCap5Ns, cfg.Namespace(), &corev1.Namespace{}, t, cfg)
	applicationSet := getOrFail(ctx, cap5ApplicationSet, applicationSetNamespace, &argo.ApplicationSet{}, t, cfg)
	cap5Quota := getOrFail(ctx, paasCap5Ns, cfg.Namespace(), &quotav1.ClusterResourceQuota{}, t, cfg)

	// ClusterResource is created with the same name as the PaaS
	assert.Equal(t, paasWithCapability5, paas.Name)

	// Paas Namespace exist
	assert.Equal(t, paasCap5Ns, namespace.Name)

	// cap5 should be enabled
	assert.Contains(t, paas.Spec.Capabilities, "cap5")

	// ApplicationSet exist
	assert.NotEmpty(t, applicationSet)

	applicationSetListEntries, appSetListEntriesError := getApplicationSetListEntries(applicationSet)

	// List entries should not be empty
	require.NoError(t, appSetListEntriesError)
	assert.Len(t, applicationSetListEntries, 1)

	// At least one JSON object should have "paas": "cap5paas"
	assert.Contains(t, applicationSetListEntries, paasWithCapability5)

	// Check whether the LabelSelector is specific to the cap5paas-cap5 namespace
	labelSelector := cap5Quota.Spec.Selector.LabelSelector
	assert.True(t, MatchLabelExists(labelSelector.MatchLabels, "q.lbl", paasCap5Ns))
	assert.False(t, MatchLabelExists(labelSelector.MatchLabels, "q.lbl", "wrong-value"))
	assert.False(t, MatchLabelExists(labelSelector.MatchLabels, "nonexistent.lbl", paasCap5Ns))

	// Quota namespace name
	assert.Equal(t, paasCap5Ns, cap5Quota.Name)

	// Cap5 quota size matches those passed in the PaaS spec
	assert.Equal(t, resource.MustParse("5"), cap5Quota.Spec.Quota.Hard[corev1.ResourceRequestsCPU])
	assert.Equal(t, resource.MustParse("6Gi"), cap5Quota.Spec.Quota.Hard[corev1.ResourceRequestsMemory])

	return ctx
}

func assertCap5Deleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaasSync(ctx, paasWithCapability5, t, cfg)

	// Quota is deleted
	var quotaList quotav1.ClusterResourceQuotaList
	if err := cfg.Client().Resources().List(ctx, &quotaList); err != nil {
		t.Fatalf("Failed to retrieve Quota list: %v", err)
	}

	// Quota list not contains paas
	assert.NotContains(t, quotaList.Items, paasCap5Ns)

	// Namespace is deleted
	var namespaceList corev1.NamespaceList
	if err := cfg.Client().Resources().List(ctx, &namespaceList); err != nil {
		t.Fatalf("Failed to retrieve Namespace list: %v", err)
	}

	// Namespace list not contains paas
	assert.NotContains(t, namespaceList.Items, paasWithCapability5)

	// ApplicationSet is deleted
	applicationSet := getOrFail(ctx, cap5ApplicationSet, applicationSetNamespace, &argo.ApplicationSet{}, t, cfg)
	applicationSetListEntries, appSetListEntriesError := getApplicationSetListEntries(applicationSet)

	// List Entries should be empty
	require.NoError(t, appSetListEntriesError)
	assert.Empty(t, applicationSetListEntries)

	return ctx
}

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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasWithCapabilitySSO   = "paasnaam"
	paasSSO                 = "paasnaam-sso"
	ssoApplicationSet       = "ssoas"
	applicationSetNamespace = "asns"
)

func TestCapabilitySSO(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: "paas-user",
		Quota:     make(quota.Quotas),
		Capabilities: api.PaasCapabilities{
			SSO: api.PaasSSO{Enabled: true},
		},
	}

	testenv.Test(
		t,
		features.New("Capability SSO").
			Setup(createPaasFn(paasWithCapabilitySSO, paasSpec)).
			Assess("is created", assertCapSSOCreated).
			Assess("is deleted when PaaS is deleted", assertCapSSODeleted).
			Teardown(teardownPaasFn(paasWithCapabilitySSO)).
			Feature(),
	)
}

func assertCapSSOCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithCapabilitySSO, t, cfg)
	namespace := getOrFail(ctx, paasWithCapabilitySSO, cfg.Namespace(), &corev1.Namespace{}, t, cfg)
	applicationSet := getOrFail(ctx, ssoApplicationSet, applicationSetNamespace, &argo.ApplicationSet{}, t, cfg)
	ssoQuota := getOrFail(ctx, paasSSO, cfg.Namespace(), &quotav1.ClusterResourceQuota{}, t, cfg)

	// ClusterResource is created with the same name as the PaaS
	assert.Equal(t, paasWithCapabilitySSO, paas.Name)

	// Paas Namespace exist
	assert.Equal(t, paasWithCapabilitySSO, namespace.Name)

	// SSO should be enabled
	assert.True(t, paas.Spec.Capabilities.SSO.Enabled)

	// ApplicationSet exist
	assert.NotEmpty(t, applicationSet)

	applicationSetListEntries, appSetListEntriesError := getApplicationSetListEntries(applicationSet)

	// List entries should not be empty
	require.NoError(t, appSetListEntriesError)
	assert.NotEmpty(t, applicationSetListEntries)

	// Flag to check if we find a JSON object
	foundNameTest := false

	for _, jsonString := range applicationSetListEntries {
		var obj map[string]interface{}
		err := json.Unmarshal([]byte(jsonString), &obj)

		// Check of json successfully unmarshalled
		require.NoError(t, err)

		// Check if the JSON object has a "paas" property with value "paasnaam"
		if paas, ok := obj["paas"]; ok && paas == paasWithCapabilitySSO {
			foundNameTest = true
		}
	}

	// At least one JSON object should have "paas": "paasnaam"
	assert.True(t, foundNameTest)

	// Check whether the LabelSelector is specific to the paasnaam-sso namespace
	labelSelector := ssoQuota.Spec.Selector.LabelSelector
	assert.True(t, MatchLabelExists(labelSelector.MatchLabels, "q.lbl", paasSSO))
	assert.False(t, MatchLabelExists(labelSelector.MatchLabels, "q.lbl", "wrong-value"))
	assert.False(t, MatchLabelExists(labelSelector.MatchLabels, "nonexistent.lbl", paasSSO))

	// Quota namespace name
	assert.Equal(t, paasSSO, ssoQuota.Name)

	// SSO quota size matches those passed in the PaaS spec
	assert.Equal(t, resource.MustParse("100m"), ssoQuota.Spec.Quota.Hard[corev1.ResourceRequestsCPU])
	assert.Equal(t, resource.MustParse("128Mi"), ssoQuota.Spec.Quota.Hard[corev1.ResourceRequestsMemory])

	return ctx
}

func assertCapSSODeleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaasSync(ctx, paasWithCapabilitySSO, t, cfg)

	// Quota is deleted
	var quotaList quotav1.ClusterResourceQuotaList
	if err := cfg.Client().Resources().List(ctx, &quotaList); err != nil {
		t.Fatalf("Failed to retrieve Quota list: %v", err)
	}

	// Quota list not contains paas
	assert.NotContains(t, quotaList.Items, paasSSO)

	// Namespace is deleted
	var namespaceList corev1.NamespaceList
	if err := cfg.Client().Resources().List(ctx, &namespaceList); err != nil {
		t.Fatalf("Failed to retrieve Namespace list: %v", err)
	}

	// Namespace list not contains paas
	assert.NotContains(t, namespaceList.Items, paasWithCapabilitySSO)

	// ApplicationSet is deleted
	applicationSet := getOrFail(ctx, ssoApplicationSet, applicationSetNamespace, &argo.ApplicationSet{}, t, cfg)
	applicationSetListEntries, appSetListEntriesError := getApplicationSetListEntries(applicationSet)

	// List Entries should be empty
	require.NoError(t, appSetListEntriesError)
	assert.Empty(t, applicationSetListEntries)

	return ctx
}

func MatchLabelExists(matchLabels map[string]string, key string, value string) bool {
	if v, exists := matchLabels[key]; exists && v == value {
		return true
	}
	return false
}

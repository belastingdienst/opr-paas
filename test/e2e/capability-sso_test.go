package e2e

import (
	"context"
	"fmt"
	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"

	"github.com/belastingdienst/opr-paas/internal/quota"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	appv1 "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const paasWithCapabilitySSO = "paasnaam"
const paasSSO = "paasnaam-sso"
const ssoApplicationSet = "ssoas"

func TestCapabilitySSO(t *testing.T) {
	capabilities := api.PaasCapabilities{
		SSO: api.PaasSSO{
			Enabled: true,
		},
	}

	paasSpec := api.PaasSpec{
		Requestor:    "paas-user",
		Quota:        make(quota.Quotas),
		Capabilities: capabilities,
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
	namespace := getNamespace(ctx, t, cfg, paasWithCapabilitySSO)
	applicationSet := getApplicationSet(ctx, t, cfg)

	fmt.Println("AppSet before deleting:", applicationSet)
	fmt.Println("=========================================")

	// ClusterResource is created with the same name as the PaaS
	assert.Equal(t, paasWithCapabilitySSO, paas.Name)

	// Paas Namespace exist
	assert.Equal(t, namespace.Name, paasWithCapabilitySSO)

	// SSO should be enabled
	assert.True(t, paas.Spec.Capabilities.SSO.Enabled)

	// ApplicationSet exist
	assert.NotEmpty(t, applicationSet)

	ssoQuota := getSsoQuota(ctx, t, cfg)
	labelSelector := ssoQuota.Spec.Selector.LabelSelector

	// Check whether the LabelSelector is specific to the paasnaam-sso namespace
	// q.lbl=paasnaam-sso should exist in MatchLabels
	assert.True(t, MatchLabelExists(labelSelector.MatchLabels, "q.lbl", paasSSO))

	// q.lbl=wrong-value should not exist in MatchLabels
	assert.False(t, MatchLabelExists(labelSelector.MatchLabels, "q.lbl", "wrong-value"))

	// nonexistent.lbl=paasnaam-sso should not exist in MatchLabels
	assert.False(t, MatchLabelExists(labelSelector.MatchLabels, "nonexistent.lbl", paasSSO))

	// Quota namespace name
	assert.Equal(t, paasSSO, ssoQuota.Name)

	// SSO quota size matches those passed in the PaaS spec
	assert.Equal(t, resource.MustParse("100m"), ssoQuota.Spec.Quota.Hard[corev1.ResourceRequestsCPU])
	assert.Equal(t, resource.MustParse("128Mi"), ssoQuota.Spec.Quota.Hard[corev1.ResourceRequestsMemory])

	return ctx
}

func assertCapSSODeleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaas(ctx, paasWithCapabilitySSO, t, cfg)

	// Quota is deleted
	var quotaList quotav1.ClusterResourceQuotaList
	if err := cfg.Client().Resources().List(ctx, &quotaList); err != nil {
		t.Fatalf("Failed to retrieve Quota list: %v", err)
	}

	assert.NotContains(t, quotaList.Items, paasSSO)

	// Namespace is deleted
	var namespaceList corev1.NamespaceList
	if err := cfg.Client().Resources().List(ctx, &namespaceList); err != nil {
		t.Fatalf("Failed to retrieve Namespace list: %v", err)
	}

	assert.NotContains(t, namespaceList.Items, paasWithCapabilitySSO)

	// ApplicationSet is deleted
	applicationSet := getApplicationSet(ctx, t, cfg)

	fmt.Println("AppSet after deleting:", applicationSet)
	fmt.Println("=========================================")

	return ctx
}

func getApplicationSet(ctx context.Context, t *testing.T, cfg *envconf.Config) appv1.ApplicationSet {
	var appSet appv1.ApplicationSet

	if err := cfg.Client().Resources().Get(ctx, ssoApplicationSet, "asns", &appSet); err != nil {
		t.Fatal(err)
	}

	return appSet
}

func getNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config, name string) corev1.Namespace {
	var ns corev1.Namespace

	if err := cfg.Client().Resources().Get(ctx, name, cfg.Namespace(), &ns); err != nil {
		t.Fatal(err)
	}

	return ns
}

func getSsoQuota(ctx context.Context, t *testing.T, cfg *envconf.Config) quotav1.ClusterResourceQuota {
	var quota quotav1.ClusterResourceQuota

	if err := cfg.Client().Resources().Get(ctx, paasSSO, cfg.Namespace(), &quota); err != nil {
		t.Fatalf("Failed to retrieve ssoQuota: %v", err)
	}

	return quota
}

func MatchLabelExists(matchLabels map[string]string, key string, value string) bool {
	if v, exists := matchLabels[key]; exists && v == value {
		return true
	}
	return false
}

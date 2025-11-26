package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/pkg/quota"
	quotav1 "github.com/openshift/api/quota/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasWithCapabilityExternal = "capexternalpaas"
	paasCapExternalNs          = "capexternalpaas-capexternal"
	capExternalApplicationSet  = "capexternalas"
)

func TestCapExternal(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: "paas-user",
		Quota:     make(quota.Quota),
		Capabilities: api.PaasCapabilities{
			"capexternal": api.PaasCapability{},
		},
	}

	testenv.Test(
		t,
		features.New("Capability External").
			Setup(createPaasFn(paasWithCapabilityExternal, paasSpec)).
			Assess("is created", assertCapExternalCreated).
			Assess("is deleted when PaaS is deleted", assertCapExternalDeleted).
			Teardown(teardownPaasFn(paasWithCapabilityExternal)).
			Feature(),
	)
}

func assertCapExternalCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithCapabilityExternal, t, cfg)

	// no namespace should be created for an external capability
	t.Log("checking if namespace is created")
	failWhenExists(ctx, paasCapExternalNs, cfg.Namespace(), &corev1.Namespace{}, t, cfg)

	t.Log("checking if clusterquota is created")
	// no quota should be created for an external capability
	failWhenExists(ctx, paasCapExternalNs, cfg.Namespace(), &quotav1.ClusterResourceQuota{}, t, cfg)

	// ClusterResource is created with the same name as the PaaS
	assert.Equal(t, paasWithCapabilityExternal, paas.Name)

	// capExternal should be enabled
	assert.Contains(t, paas.Spec.Capabilities, "capexternal")

	return ctx
}

func assertCapExternalDeleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaasSync(ctx, paasWithCapabilityExternal, t, cfg)

	// Namespace is deleted
	var namespaceList corev1.NamespaceList
	if err := cfg.Client().Resources().List(ctx, &namespaceList); err != nil {
		t.Fatalf("Failed to retrieve Namespace list: %v", err)
	}

	// Namespace list not contains paas
	assert.NotContains(t, namespaceList.Items, paasWithCapabilityExternal)

	return ctx
}

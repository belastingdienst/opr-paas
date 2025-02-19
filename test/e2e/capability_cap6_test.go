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
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasWithCapability6 = "cap6paas"
	cap6Name            = "cap6"
	paasCap6Ns          = "cap6paas-cap6"
	paasCap6CRQ         = "paas-cap6"
	cap6ApplicationSet  = "cap6as"
)

func TestCapabilityCap6(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: "paas-user",
		Quota:     make(quota.Quota),
		Capabilities: api.PaasCapabilities{
			cap6Name: api.PaasCapability{Enabled: true},
		},
	}

	testenv.Test(
		t,
		features.New("Capability 6").
			Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				// Update the active PaasConfig to configure cap6
				var paasConfig api.PaasConfig
				err := cfg.Client().Resources().Get(ctx, "paas-config", "", &paasConfig)
				require.NoError(t, err, "Failed to get PaasConfig")

				paasConfig.Spec.Capabilities[cap6Name] = api.ConfigCapability{
					AppSet: cap6ApplicationSet,
					QuotaSettings: api.ConfigQuotaSettings{
						DefQuota: map[corev1.ResourceName]resourcev1.Quantity{},
					},
				}

				err = cfg.Client().Resources().Update(ctx, &paasConfig)
				require.NoError(t, err, "Failed to update PaasConfig")

				return ctx
			}).
			Setup(createPaasWithCondFn(paasWithCapability6, paasSpec, api.TypeHasErrorsPaas)).
			Assess("is created", assertCap6NoUnwantedArtifacts).
			Assess("is deleted when PaaS is deleted", assertCap6Deleted).
			Teardown(teardownPaasFn(paasWithCapability6)).
			Feature(),
	)
}

func assertCap6NoUnwantedArtifacts(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithCapability6, t, cfg)
	failWhenExists(ctx, cap6ApplicationSet, applicationSetNamespace, &argo.ApplicationSet{}, t, cfg)
	failWhenExists(ctx, paasCap6CRQ, cfg.Namespace(), &quotav1.ClusterResourceQuota{}, t, cfg)

	// Paas has correct name
	assert.Equal(t, paasWithCapability6, paas.Name)

	return ctx
}

func assertCap6Deleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaasSync(ctx, paasWithCapability6, t, cfg)

	// Quota is deleted
	var quotaList quotav1.ClusterResourceQuotaList
	if err := cfg.Client().Resources().List(ctx, &quotaList); err != nil {
		t.Fatalf("Failed to retrieve Quota list: %v", err)
	}

	// Quota list not contains paas
	assert.NotContains(t, quotaList.Items, paasCap6Ns)

	// Namespace is deleted
	var namespaceList corev1.NamespaceList
	if err := cfg.Client().Resources().List(ctx, &namespaceList); err != nil {
		t.Fatalf("Failed to retrieve Namespace list: %v", err)
	}

	// Namespace list not contains paas
	assert.NotContains(t, namespaceList.Items, paasWithCapability6)

	return ctx
}

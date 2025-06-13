package e2e

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"

	api "github.com/belastingdienst/opr-paas/v2/api/v1alpha1"
	quotav1 "github.com/openshift/api/quota/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const paasWithQuota = "paas-with-quota"

func TestClusterResourceQuota(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: "paas-user",
		Quota: map[corev1.ResourceName]resource.Quantity{
			"cpu":    resource.MustParse("200m"),
			"memory": resource.MustParse("256Mi"),
		},
	}

	testenv.Test(
		t,
		features.New("ClusterResourceQuota").
			Setup(createPaasFn(paasWithQuota, paasSpec)).
			Assess("is created", assertCRQCreated).
			Assess("is updated", assertCRQUpdated).
			Assess("is deleted when Paas is deleted", assertCRQDeleted).
			Teardown(teardownPaasFn(paasWithQuota)).
			Feature(),
	)
}

func assertCRQCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	crq := getCRQ(ctx, t, cfg)

	// ClusterResourceQuota is created with the same name as the Paas
	assert.Equal(t, paasWithQuota, crq.Name)
	// The label selector matches the Paas name
	assert.Equal(t, paasWithQuota, crq.Spec.Selector.LabelSelector.MatchLabels["q.lbl"])
	// The quota size matches those passed in the Paas spec
	assert.Equal(t, resource.MustParse("200m"), *crq.Spec.Quota.Hard.Cpu())
	assert.Equal(t, resource.MustParse("256Mi"), *crq.Spec.Quota.Hard.Memory())

	return ctx
}

func assertCRQUpdated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithQuota, t, cfg)

	paas.Spec.Quota = map[corev1.ResourceName]resource.Quantity{
		"cpu":    resource.MustParse("100m"),
		"memory": resource.MustParse("128Mi"),
	}

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	crq := getCRQ(ctx, t, cfg)

	assert.Equal(t, resource.MustParse("100m"), *crq.Spec.Quota.Hard.Cpu())
	assert.Equal(t, resource.MustParse("128Mi"), *crq.Spec.Quota.Hard.Memory())

	return ctx
}

func assertCRQDeleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaasSync(ctx, paasWithQuota, t, cfg)
	crqs := listOrFail(ctx, "", &quotav1.ClusterResourceQuotaList{}, t, cfg)

	assert.Empty(t, crqs.Items)

	return ctx
}

func getCRQ(ctx context.Context, t *testing.T, cfg *envconf.Config) *quotav1.ClusterResourceQuota {
	return getOrFail(ctx, paasWithQuota, cfg.Namespace(), &quotav1.ClusterResourceQuota{}, t, cfg)
}

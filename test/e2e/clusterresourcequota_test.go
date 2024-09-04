package e2e

import (
	"context"
	"testing"

	"github.com/belastingdienst/opr-paas/internal/quota"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	quotav1 "github.com/openshift/api/quota/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const paasWithQuota = "paas-with-quota"

func TestClusterResourceQuota(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: "paas-user",
		Quota: quota.NewQuota(map[string]string{
			"cpu":    "200m",
			"memory": "256Mi",
		}),
	}

	testenv.Test(
		t,
		features.New("ClusterResourceQuota").
			Setup(createPaasFn(paasWithQuota, paasSpec)).
			Assess("is created", assertCRQCreated).
			Assess("is updated", assertCRQUpdated).
			Assess("is deleted when PaaS is deleted", assertCRQDeleted).
			Feature(),
	)
}

func assertCRQCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	crq := getCRQ(ctx, t, cfg)

	assert.Equal(t, paasWithQuota, crq.Name)
	assert.Equal(t, resource.MustParse("200m"), *crq.Spec.Quota.Hard.Cpu())
	assert.Equal(t, resource.MustParse("256Mi"), *crq.Spec.Quota.Hard.Memory())

	return ctx
}

func assertCRQUpdated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	var paas api.Paas

	if err := cfg.Client().Resources().Get(ctx, paasWithQuota, cfg.Namespace(), &paas); err != nil {
		t.Fatalf("Failed to retrieve PaaS: %v", err)
	}

	paas.Spec.Quota = quota.NewQuota(map[string]string{
		"cpu":    "100m",
		"memory": "128Mi",
	})

	if err := cfg.Client().Resources().Update(ctx, &paas); err != nil {
		t.Fatalf("Failed to update PaaS resource: %v", err)
	}

	waitForOperator()

	crq := getCRQ(ctx, t, cfg)

	assert.Equal(t, resource.MustParse("100m"), *crq.Spec.Quota.Hard.Cpu())
	assert.Equal(t, resource.MustParse("128Mi"), *crq.Spec.Quota.Hard.Memory())

	return ctx
}

func assertCRQDeleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := &api.Paas{ObjectMeta: metav1.ObjectMeta{Name: paasWithQuota}}

	if err := cfg.Client().Resources().Delete(ctx, paas); err != nil {
		t.Fatalf("Failed to delete PaaS: %v", err)
	}

	waitForOperator()

	var crqs quotav1.ClusterResourceQuotaList

	if err := cfg.Client().Resources().List(ctx, &crqs); err != nil {
		t.Fatalf("Failed to retrieve list of ClusterResourceQuotas: %v", err)
	}

	assert.Empty(t, 0, len(crqs.Items))

	return ctx
}

func getCRQ(ctx context.Context, t *testing.T, cfg *envconf.Config) quotav1.ClusterResourceQuota {
	var crq quotav1.ClusterResourceQuota

	if err := cfg.Client().Resources().Get(ctx, paasWithQuota, cfg.Namespace(), &crq); err != nil {
		t.Fatalf("Failed to retrieve ClusterResourceQuota: %v", err)
	}

	return crq
}

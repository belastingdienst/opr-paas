package e2e

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasName   = "a-paas"
	paasNsName = "a-paasns"
)

func TestPaasNS(t *testing.T) {
	testenv.Test(
		t,
		features.New("PaasNS").
			Assess("PaasNS deletion", assertPaasNSDeletion).
			Assess("PaasNS creation without linked Paas", assertPaasNSCreatedWithoutPaas).
			Assess("PaasNS creation with unlinked Paas", assertPaasNSCreatedWithUnlinkedPaas).
			Assess("PaasNS creation with linked Paas", assertPaasNSCreated).
			Feature(),
	)
}

func assertPaasNSDeletion(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: cfg.Namespace(),
		},
		Spec: api.PaasNSSpec{Paas: paasName},
	}

	// create basic paasns
	pnsCreatePaasNS(ctx, t, cfg, paasNs)

	// remove it immediately
	pnsDeletePaasNS(ctx, t, cfg, paasNs)

	// check that we cannot get the paasns because we deleted it
	_, errPaasNS := pnsGetPaasNS(ctx, cfg, paasNsName, cfg.Namespace())
	require.Error(t, errPaasNS)

	return ctx
}

func assertPaasNSCreatedWithoutPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: cfg.Namespace(),
		},
		Spec: api.PaasNSSpec{Paas: paasName}, // this paas does not exist but we do reference it
	}

	// create paasns including reference to non-existent paas
	pnsCreatePaasNS(ctx, t, cfg, paasNs)

	// check that the referenced paas hasn't been created on-the-fly
	err := cfg.Client().Resources().Get(ctx, paasName, cfg.Namespace(), &api.Paas{})
	require.Error(t, err)

	// check that the paasns has been created but also contains an error status message
	fetchedPaasNS, _ := pnsGetPaasNS(ctx, cfg, paasNsName, cfg.Namespace())
	assert.True(t, meta.IsStatusConditionPresentAndEqual(fetchedPaasNS.Status.Conditions, api.TypeReadyPaasNs, metav1.ConditionFalse))
	assert.True(t, meta.IsStatusConditionPresentAndEqual(fetchedPaasNS.Status.Conditions, api.TypeHasErrorsPaasNs, metav1.ConditionTrue))
	foundCondition := meta.FindStatusCondition(paasNs.Status.Conditions, api.TypeHasErrorsPaasNs)
	assert.Contains(t, foundCondition.Message, "cannot find Paas a-paas")

	// cleanup
	pnsDeletePaasNS(ctx, t, cfg, paasNs)

	return ctx
}

func assertPaasNSCreatedWithUnlinkedPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	// setup: create paas to reference from paasns
	paas := &api.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: "new-paas",
		},
		Spec: api.PaasSpec{
			Quota: map[corev1.ResourceName]resource.Quantity{
				"cpu":    resource.MustParse("2"),
				"memory": resource.MustParse("2Gi"),
			},
		},
	}

	require.NoError(t, createPaasSyncSuccess(ctx, cfg, paas))

	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: cfg.Namespace(),
		},
		Spec: api.PaasNSSpec{Paas: "new-paas"},
	}

	pnsCreatePaasNS(ctx, t, cfg, paasNs)

	// check that the paasns has been created but also contains an error status message
	fetchedPaasNS, _ := pnsGetPaasNS(ctx, cfg, paasNsName, cfg.Namespace())
	assert.True(t, meta.IsStatusConditionPresentAndEqual(fetchedPaasNS.Status.Conditions, api.TypeReadyPaasNs, metav1.ConditionFalse))
	assert.True(t, meta.IsStatusConditionPresentAndEqual(fetchedPaasNS.Status.Conditions, api.TypeHasErrorsPaasNs, metav1.ConditionTrue))
	foundCondition := meta.FindStatusCondition(paasNs.Status.Conditions, api.TypeHasErrorsPaasNs)
	assert.Contains(t, foundCondition.Message, "not in the list of namespaces")

	// cleanup
	deletePaasSync(ctx, "new-paas", t, cfg)
	pnsDeletePaasNS(ctx, t, cfg, paasNs)

	return ctx
}

func assertPaasNSCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	thisPaas := "this-paas"
	thisNamespace := "this-namespace"
	generatedName := thisPaas + "-" + thisNamespace

	// setup: create paas to link to
	paas := &api.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: thisPaas,
		},
		Spec: api.PaasSpec{
			Namespaces: []string{thisNamespace}, // define suffixes to use for namespace names
			Quota: map[corev1.ResourceName]resource.Quantity{
				"cpu":    resource.MustParse("2"),
				"memory": resource.MustParse("2Gi"),
			},
		},
	}

	require.NoError(t, createPaasSyncSuccess(ctx, cfg, paas))

	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: generatedName,
		},
		Spec: api.PaasNSSpec{Paas: thisPaas},
	}

	pnsCreatePaasNS(ctx, t, cfg, paasNs)

	// check that the paasns has been created and is linked to the correct paas
	fetchedPaasNS, _ := pnsGetPaasNS(ctx, cfg, paasNsName, generatedName)

	linkedPaas := fetchedPaasNS.Spec.Paas
	assert.Equal(t, linkedPaas, thisPaas)

	// check that there are no errors
	assert.True(t, meta.IsStatusConditionPresentAndEqual(fetchedPaasNS.Status.Conditions, api.TypeReadyPaasNs, metav1.ConditionTrue))
	assert.True(t, meta.IsStatusConditionPresentAndEqual(fetchedPaasNS.Status.Conditions, api.TypeHasErrorsPaasNs, metav1.ConditionFalse))
	foundCondition := meta.FindStatusCondition(paasNs.Status.Conditions, api.TypeHasErrorsPaasNs)
	assert.Equal(t, "Reconciled (a-paasns) successfully", foundCondition.Message)

	// cleanup
	deletePaasSync(ctx, thisPaas, t, cfg)
	pnsDeletePaasNS(ctx, t, cfg, paasNs)

	return ctx
}

func pnsCreatePaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasns *api.PaasNS) {
	require.NoError(t, cfg.Client().Resources().Create(ctx, paasns), "failed to create PaasNS "+paasns.GetName())

	waitUntilPaasNSExists := conditions.New(cfg.Client().Resources()).ResourceMatch(paasns, func(obj k8s.Object) bool {
		return obj.(*api.PaasNS).Name == paasns.Name
	})
	require.NoError(t, waitForDefaultOpts(ctx, waitUntilPaasNSExists))
}

func pnsGetPaasNS(ctx context.Context, cfg *envconf.Config, paasnsName string, namespace string) (paasns api.PaasNS, err error) {
	err = cfg.Client().Resources().Get(ctx, paasnsName, namespace, &paasns)
	return paasns, err
}

func pnsDeletePaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasns *api.PaasNS) {
	require.NoError(t, deleteResourceSync(ctx, cfg, paasns))
}

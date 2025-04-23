package e2e

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasNsPaasName = "paasns-paas"
	paasNsName     = "paasns"
)

func TestPaasNS(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: "paasns-requestor",
		Quota:     make(quota.Quota),
	}
	testenv.Test(
		t,
		features.New("PaasNS").
			Assess("PaasNS deletion", assertPaasNSDeletion).
			Setup(createPaasFn(paasNsPaasName, paasSpec)).
			Assess("PaasNS creation with linked Paas", assertPaasNSCreated).
			Assess("PaasNS deletion", assertPaasNSDeletion).
			Feature(),
	)
}

func assertPaasNSDeletion(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: paasNsPaasName,
		},
		Spec: api.PaasNSSpec{Paas: paasNsPaasName},
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
			Requestor:  paasRequestor,
			Namespaces: []string{thisNamespace}, // define suffixes to use for namespace names
			Quota: map[corev1.ResourceName]resource.Quantity{
				"cpu":    resource.MustParse("2"),
				"memory": resource.MustParse("2Gi"),
			},
		},
	}

	require.NoError(t, createSync(ctx, cfg, paas, api.TypeReadyPaas))

	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: generatedName,
		},
		Spec: api.PaasNSSpec{Paas: thisPaas},
	}

	pnsCreatePaasNS(ctx, t, cfg, paasNs)

	fetchedPaas := getPaas(ctx, thisPaas, t, cfg)

	// check that there are no errors
	assert.True(
		t,
		meta.IsStatusConditionPresentAndEqual(
			fetchedPaas.Status.Conditions,
			api.TypeReadyPaas,
			metav1.ConditionTrue,
		),
	)
	assert.True(
		t,
		meta.IsStatusConditionPresentAndEqual(
			fetchedPaas.Status.Conditions,
			api.TypeHasErrorsPaas,
			metav1.ConditionFalse,
		),
	)
	foundCondition := meta.FindStatusCondition(paas.Status.Conditions, api.TypeHasErrorsPaas)
	assert.Equal(t, fmt.Sprintf("Reconciled (%s) successfully", thisPaas), foundCondition.Message)

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

func pnsGetPaasNS(
	ctx context.Context,
	cfg *envconf.Config,
	paasnsName string,
	namespace string,
) (paasns api.PaasNS, err error) {
	err = cfg.Client().Resources().Get(ctx, paasnsName, namespace, &paasns)
	return paasns, err
}

func pnsDeletePaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasns *api.PaasNS) {
	require.NoError(t, deleteResourceSync(ctx, cfg, paasns))
}

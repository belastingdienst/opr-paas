package e2e

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	api "github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasNsName = "paasns"
)

func TestPaasNS(t *testing.T) {
	testenv.Test(
		t,
		features.New("PaasNS").
			Assess("PaasNS creation with linked Paas", assertPaasNSCreated).
			Feature(),
	)
}

func assertPaasNSCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	const (
		thisPaas      = "this-paas"
		thisNamespace = "this-namespace"
		generatedName = thisPaas + "-" + thisNamespace
	)

	// setup: create paas to link to
	paas := &api.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: thisPaas,
		},
		Spec: api.PaasSpec{
			Requestor:  paasRequestor,
			Namespaces: api.PaasNamespaces{thisNamespace: api.PaasNamespace{}},
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

	_ = getOrFail(ctx, fmt.Sprintf("%s-%s", thisPaas, paasNsName), cfg.Namespace(), &corev1.Namespace{}, t, cfg)

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
	failWhenExists(ctx, fmt.Sprintf("%s-%s", generatedName, paasName), cfg.Namespace(), &corev1.Namespace{}, t, cfg)

	return ctx
}

func pnsCreatePaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasns *api.PaasNS) {
	require.NoError(t, cfg.Client().Resources().Create(ctx, paasns), "failed to create PaasNS "+paasns.GetName())

	waitUntilPaasNSExists := conditions.New(cfg.Client().Resources()).ResourceMatch(paasns, func(obj k8s.Object) bool {
		return obj.(*api.PaasNS).Name == paasns.Name
	})
	require.NoError(t, waitForDefaultOpts(ctx, waitUntilPaasNSExists))
}

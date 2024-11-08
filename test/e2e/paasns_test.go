package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
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
	pnsCreatePaasNS(ctx, t, cfg, *paasNs)

	// remove it immediately
	pnsDeletePaasNS(ctx, t, cfg, *paasNs)

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
	pnsCreatePaasNS(ctx, t, cfg, *paasNs)

	// check that the referenced paas hasn't been created on-the-fly
	_, errPaas := pnsGetPaas(ctx, cfg)
	require.Error(t, errPaas)

	// check that the paasns has been created but also contains an error status message
	fetchedPaasNS, _ := pnsGetPaasNS(ctx, cfg, paasNsName, cfg.Namespace())
	errMsg := fetchedPaasNS.Status.Messages[0]
	assert.Contains(t, errMsg, "cannot find PaaS")

	// cleanup
	pnsDeletePaasNS(ctx, t, cfg, *paasNs)

	return ctx
}

func assertPaasNSCreatedWithUnlinkedPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	thisPaasName := "new-paas"
	// setup: create paas to reference from paasns
	paasSpec := api.PaasSpec{
		Quota: quota.NewQuota(map[string]string{
			"cpu":    "2",
			"memory": "2Gi",
		}),
	}

	createPaasFn(thisPaasName, paasSpec)

	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: cfg.Namespace(),
		},
		Spec: api.PaasNSSpec{Paas: "new-paas"},
	}

	pnsCreatePaasNS(ctx, t, cfg, *paasNs)

	// check that the paasns has been created but also contains an error status message
	fetchedPaasNS, _ := pnsGetPaasNS(ctx, cfg, paasNsName, cfg.Namespace())
	errMsg := fetchedPaasNS.Status.Messages[0]
	assert.Contains(t, errMsg, "not in the list of namespaces")

	// cleanup
	deletePaasSync(ctx, thisPaasName, t, cfg)
	pnsDeletePaasNS(ctx, t, cfg, *paasNs)

	return ctx
}

func assertPaasNSCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	thisPaas := "this-paas"
	thisNamespace := "this-namespace"
	generatedName := thisPaas + "-" + thisNamespace

	// setup: create paas to link to
	paasSpec := api.PaasSpec{
		Namespaces: []string{thisNamespace}, // define suffixes to use for namespace names
		Quota: quota.NewQuota(map[string]string{
			"cpu":    "2",
			"memory": "2Gi",
		}),
	}

	createPaasFn(thisPaas, paasSpec)

	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: generatedName,
		},
		Spec: api.PaasNSSpec{Paas: thisPaas},
	}

	pnsCreatePaasNS(ctx, t, cfg, *paasNs)

	// check that the paasns has been created and is linked to the correct paas
	fetchedPaasNS, _ := pnsGetPaasNS(ctx, cfg, paasNsName, generatedName)

	linkedPaas := fetchedPaasNS.Spec.Paas
	assert.Equal(t, linkedPaas, thisPaas)

	// check that there are no errors
	assert.NotContains(t, fetchedPaasNS.Status.Messages, "ERROR")

	// cleanup
	deletePaasSync(ctx, thisPaas, t, cfg)
	pnsDeletePaasNS(ctx, t, cfg, *paasNs)

	return ctx
}

func pnsCreatePaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasns api.PaasNS) api.PaasNS {
	if err := cfg.Client().Resources().Create(ctx, &paasns); err != nil {
		t.Fatalf("Failed to create PaasNS: %v", err)
	}

	waitUntilPaasNSExists := wait.For(conditions.New(cfg.Client().Resources()).ResourceMatch(&paasns, func(obj k8s.Object) bool {
		return obj.(*api.PaasNS).Name == paasns.Name
	}))

	if waitUntilPaasNSExists != nil {
		t.Fatalf("PaasNS creation was fired successfully but PaasNS wasn't found to exist: %v", waitUntilPaasNSExists)
	}

	return paasns
}

func pnsGetPaasNS(ctx context.Context, cfg *envconf.Config, paasnsName string, namespace string) (paasns api.PaasNS, err error) {
	err = cfg.Client().Resources().Get(ctx, paasnsName, namespace, &paasns)
	return paasns, err
}

func pnsDeletePaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasns api.PaasNS) {
	if err := cfg.Client().Resources().Delete(ctx, &paasns); err != nil {
		t.Fatalf("Failed to delete PaasNS: %v", err)
	}

	waitUntilPaasNSDeleted := wait.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(&paasns))

	if waitUntilPaasNSDeleted != nil {
		t.Fatalf("PaasNS deletion was initiated but we never got a v1.StatusReasonNotFound error: %v", waitUntilPaasNSDeleted)
	}
}

//func pnsCreatePaas(ctx context.Context, t *testing.T, cfg *envconf.Config, paas api.Paas) api.Paas {
//	if err := cfg.Client().Resources().Create(ctx, &paas); err != nil {
//		t.Fatalf("Failed to create Paas: %v", err)
//	}
//
//	waitUntilPaasExists := wait.For(conditions.New(cfg.Client().Resources()).ResourceMatch(&paas, func(obj k8s.Object) bool {
//		return obj.(*api.Paas).Name == paas.Name
//	}))
//
//	if waitUntilPaasExists != nil {
//		t.Fatalf("Paas creation was fired successfully but Paas wasn't found to exist: %v", waitUntilPaasExists)
//	}
//
//	return paas
//}

func pnsGetPaas(ctx context.Context, cfg *envconf.Config) (paas api.Paas, err error) {
	err = cfg.Client().Resources().Get(ctx, paasName, cfg.Namespace(), &paas)
	return paas, err
}

//func pnsDeletePaas(ctx context.Context, t *testing.T, cfg *envconf.Config, paas api.Paas) {
//	if err := cfg.Client().Resources().Delete(ctx, &paas); err != nil {
//		t.Fatalf("Failed to delete Paas: %v", err)
//	}
//
//	waitUntilPaasDeleted := wait.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(&paas))
//
//	if waitUntilPaasDeleted != nil {
//		t.Fatalf("Paas deletion was initiated but we never got a v1.StatusReasonNotFound error: %v", waitUntilPaasDeleted)
//	}
//}

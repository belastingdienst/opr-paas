package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// give cluster some time to create resources otherwise asserts fire too soon
	// possibly better solved by separating into Setup() and Assess() steps (refactor)
	// or we could maybe use sigs.k8s.io/e2e-framework/klient/wait
	waitForOperator()

	// remove it immediately
	pnsDeletePaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	// check that we cannot get the paasns because we deleted it
	_, errPaasNS := func() (paasns api.PaasNS, err error) {
		var namespace string = cfg.Namespace()
		return pnsGetPaasNS(ctx, cfg, paasNsName, namespace)
	}()
	waitForOperator()
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
	waitForOperator()

	// check that the referenced paas hasn't been created on-the-fly
	var errPaas error
	_, errPaas = pnsGetPaas(ctx, cfg)
	require.Error(t, errPaas)

	// check that the paasns has been created but also contains an error status message
	fetchedPaasNS, _ := func() (paasns api.PaasNS, err error) {
		var namespace string = cfg.Namespace()
		return pnsGetPaasNS(ctx, cfg, paasNsName, namespace)
	}()
	errMsg := fetchedPaasNS.Status.Messages[0]
	assert.Contains(t, errMsg, "cannot find PaaS")

	// cleanup
	pnsDeletePaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	return ctx
}

func assertPaasNSCreatedWithUnlinkedPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	// setup: create paas to reference from paasns
	paas := &api.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: "new-paas",
		},
		Spec: api.PaasSpec{
			Quota: quota.NewQuota(map[string]string{
				"cpu":    "2",
				"memory": "2Gi",
			}),
		},
	}

	pnsCreatePaas(ctx, t, cfg, *paas)
	waitForOperator()

	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: cfg.Namespace(),
		},
		Spec: api.PaasNSSpec{Paas: "new-paas"},
	}

	pnsCreatePaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	// check that the paasns has been created but also contains an error status message
	fetchedPaasNS, _ := func() (paasns api.PaasNS, err error) {
		var namespace string = cfg.Namespace()
		return pnsGetPaasNS(ctx, cfg, paasNsName, namespace)
	}()
	errMsg := fetchedPaasNS.Status.Messages[0]
	assert.Contains(t, errMsg, "not in the list of namespaces")

	// cleanup
	pnsDeletePaas(ctx, t, cfg, *paas)
	pnsDeletePaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

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
			Quota: quota.NewQuota(map[string]string{
				"cpu":    "2",
				"memory": "2Gi",
			}),
		},
	}

	pnsCreatePaas(ctx, t, cfg, *paas)
	waitForOperator()

	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: generatedName,
		},
		Spec: api.PaasNSSpec{Paas: thisPaas},
	}

	pnsCreatePaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	// check that the paasns has been created and is linked to the correct paas
	fetchedPaasNS, _ := pnsGetPaasNS(ctx, cfg, paasNsName, generatedName)
	waitForOperator()

	linkedPaas := fetchedPaasNS.Spec.Paas
	assert.Equal(t, linkedPaas, thisPaas)

	// check that there are no errors
	assert.NotContains(t, fetchedPaasNS.Status.Messages, "ERROR")

	// cleanup
	pnsDeletePaas(ctx, t, cfg, *paas)
	pnsDeletePaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	return ctx
}

func pnsCreatePaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasns api.PaasNS) api.PaasNS {
	if err := cfg.Client().Resources().Create(ctx, &paasns); err != nil {
		t.Fatalf("Failed to create PaasNS: %v", err)
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
}

func pnsCreatePaas(ctx context.Context, t *testing.T, cfg *envconf.Config, paas api.Paas) api.Paas {
	if err := cfg.Client().Resources().Create(ctx, &paas); err != nil {
		t.Fatalf("Failed to create Paas: %v", err)
	}

	return paas
}

func pnsGetPaas(ctx context.Context, cfg *envconf.Config) (paas api.Paas, err error) {
	err = cfg.Client().Resources().Get(ctx, paasName, cfg.Namespace(), &paas)
	return paas, err
}

func pnsDeletePaas(ctx context.Context, t *testing.T, cfg *envconf.Config, paas api.Paas) {
	if err := cfg.Client().Resources().Delete(ctx, &paas); err != nil {
		t.Fatalf("Failed to delete Paas: %v", err)
	}
}

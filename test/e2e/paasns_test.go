package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const paasName = "a-paas"
const paasNsName = "a-paasns"

// Consider reorganising into 3 feature segments,
// possibly removing the need for wait/sleep

// Features:
// 1 - PaasNS creation without linked Paas
// 2 - PaasNS creation with linked Paas
// 3 - PaasNS deletion

// func PaasNSCreationWithoutLinkedPaas(t *testing.T) {
// 	badPaasNS := &api.PaasNS{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      paasNsName,
// 			Namespace: "paas-e2e",
// 		},
// 		Spec: api.PaasNSSpec{Paas: paasName}, // this paas does not exist but we do reference it
// 	}

// Should we not use the e2e package setup-feature-teardown steps?
// 	testenv.Test(
// 		t,
// 		features.New("PaasNS creation without linked Paas").
// 		Setup().
// 		Feature(),
// 	)
// }

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
	createPaasNS(ctx, t, cfg, *paasNs)

	// give cluster some time to create resources otherwise asserts fire too soon
	// possibly better solved by seperating into Setup() and Assess() steps (refactor)
	// or we could maybe use sigs.k8s.io/e2e-framework/klient/wait
	waitForOperator()

	// remove it immediately
	deletePaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	// check that we cannot get the paasns because we deleted it
	_, errPaasNS := getPaasNS(ctx, t, cfg, paasNsName, cfg.Namespace())
	assert.Error(t, errPaasNS)
	waitForOperator()

	return ctx
}

func assertPaasNSCreatedWithoutPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: "paas-e2e",
		},
		Spec: api.PaasNSSpec{Paas: "a-paas"}, // this paas does not exist but we do reference it
	}

	// create paasns including reference to non-existent paas
	createPaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	// check that the referenced paas hasn't been created on-the-fly
	_, errPaas := getPaas(ctx, t, cfg)
	assert.Error(t, errPaas)

	// check that the paasns has been created but also contains an error status message
	fetchedPaasNS, _ := getPaasNS(ctx, t, cfg, paasNsName, cfg.Namespace())
	var errMsg = fetchedPaasNS.Status.Messages[0]
	assert.Contains(t, errMsg, "cannot find PaaS")

	// cleanup
	deletePaasNS(ctx, t, cfg, *paasNs)
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

	createPaas(ctx, t, cfg, *paas)
	waitForOperator()

	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: "paas-e2e",
		},
		Spec: api.PaasNSSpec{Paas: "new-paas"},
	}

	createPaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	// check that the paasns has been created but also contains an error status message
	fetchedPaasNS, _ := getPaasNS(ctx, t, cfg, paasNsName, cfg.Namespace())
	var errMsg = fetchedPaasNS.Status.Messages[0]
	assert.Contains(t, errMsg, "not in the list of namespaces")

	// cleanup
	deletePaas(ctx, t, cfg, *paas)
	deletePaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	return ctx
}

func assertPaasNSCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	var thisPaas = "this-paas"
	var thisNamespace = "this-namespace"
	var generatedName = thisPaas + "-" + thisNamespace

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

	createPaas(ctx, t, cfg, *paas)
	waitForOperator()

	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: generatedName,
			// Namespace: "paas-e2e",
		},
		Spec: api.PaasNSSpec{Paas: thisPaas},
	}

	createPaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	// check that the paasns has been created and is linked to the correct paas
	fetchedPaasNS, _ := getPaasNS(ctx, t, cfg, paasNsName, generatedName)
	waitForOperator()

	var linkedPaas = fetchedPaasNS.Spec.Paas
	assert.Equal(t, linkedPaas, thisPaas)

	// check that there are no errors
	assert.NotContains(t, fetchedPaasNS.Status.Messages, "ERROR")

	// cleanup
	deletePaas(ctx, t, cfg, *paas)
	deletePaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	return ctx
}

func createPaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasns api.PaasNS) api.PaasNS {

	if err := cfg.Client().Resources().Create(ctx, &paasns); err != nil {
		t.Fatalf("Failed to create PaasNS: %v", err)
	}

	return paasns
}

func getPaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasnsName string, namespace string) (paasns api.PaasNS, err error) {
	err = cfg.Client().Resources().Get(ctx, paasnsName, namespace, &paasns)
	return paasns, err
}

func deletePaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasns api.PaasNS) {
	if err := cfg.Client().Resources().Delete(ctx, &paasns); err != nil {
		t.Fatalf("Failed to delete PaasNS: %v", err)
	}
}

func createPaas(ctx context.Context, t *testing.T, cfg *envconf.Config, paas api.Paas) api.Paas {
	if err := cfg.Client().Resources().Create(ctx, &paas); err != nil {
		t.Fatalf("Failed to create Paas: %v", err)
	}

	return paas
}

func getPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) (paas api.Paas, err error) {
	err = cfg.Client().Resources().Get(ctx, paasName, cfg.Namespace(), &paas)
	return paas, err
}

func deletePaas(ctx context.Context, t *testing.T, cfg *envconf.Config, paas api.Paas) {
	if err := cfg.Client().Resources().Delete(ctx, &paas); err != nil {
		t.Fatalf("Failed to delete Paas: %v", err)
	}
}

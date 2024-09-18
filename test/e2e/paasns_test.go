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

const paasNamespace = "paas-e2e"
const paasNsName = "a-paasns"
const paasName = "a-paas"

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
			// Setup(createPaasFn(paasWithQuota, paasSpec)).
			Assess("PaasNS creation without linked Paas", assertPaasNSCreatedWithoutPaas).
			Assess("PaasNS creation with linked Paas", assertPaasNSCreated).
			// Assess("PaasNS deletion", assertPaasNSDeletion).
			Feature(),
	)
}

func assertPaasNSCreatedWithoutPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: "paas-e2e",
		},
		Spec: api.PaasNSSpec{Paas: paasName}, // this paas does not exist but we do reference it
	}

	// create paasns including reference to non-existent paas
	createPaasNS(ctx, t, cfg, *paasNs)

	// give cluster some time to create resources otherwise asserts fire too soon
	// possibly better solved by seperating into Setup() and Assess() steps (refactor)
	// or we could maybe use sigs.k8s.io/e2e-framework/klient/wait
	waitForOperator()

	// check that the referenced paas hasn't been created on-the-fly
	_, errPaas := getPaas(ctx, t, cfg)
	assert.Error(t, errPaas)

	// check that the paasns has been created but also contains an error status message
	fetchedPaasNS := getPaasNS(ctx, t, cfg)
	var errMsg = fetchedPaasNS.Status.Messages[0]
	assert.Contains(t, errMsg, "cannot find PaaS")

	// cleanup
	deletePaasNS(ctx, t, cfg, *paasNs)

	return ctx
}

func assertPaasNSCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	// var quota = api.PaasSpec.Quota

	// setup: create paas to link to
	paas := &api.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasName,
			Namespace: paasNamespace,
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
			Namespace: paasNamespace,
		},
		Spec: api.PaasNSSpec{Paas: paasName},
	}

	createPaasNS(ctx, t, cfg, *paasNs)
	waitForOperator()

	// check that the paasns has been created and is linked to the correct paas
	fetchedPaasNS := getPaasNS(ctx, t, cfg)
	var linkedPaas = fetchedPaasNS.Spec.Paas
	assert.Equal(t, linkedPaas, paasName)

	// cleanup
	deletePaas(ctx, t, cfg, *paas)
	deletePaasNS(ctx, t, cfg, *paasNs)

	return ctx
}

func createPaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasns api.PaasNS) api.PaasNS {

	if err := cfg.Client().Resources().Create(ctx, &paasns); err != nil {
		t.Fatalf("Failed to create PaasNS: %v", err)
	}

	return paasns
}

func getPaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config) api.PaasNS {
	var paasns api.PaasNS

	if err := cfg.Client().Resources().Get(ctx, paasNsName, cfg.Namespace(), &paasns); err != nil {
		t.Fatalf("Failed to retrieve PaasNS: %v", err)
	}

	return paasns
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
	err = cfg.Client().Resources().Get(ctx, paasNsName, cfg.Namespace(), &paas)
	return paas, err
}

func deletePaas(ctx context.Context, t *testing.T, cfg *envconf.Config, paas api.Paas) {
	if err := cfg.Client().Resources().Delete(ctx, &paas); err != nil {
		t.Fatalf("Failed to delete Paas: %v", err)
	}
}

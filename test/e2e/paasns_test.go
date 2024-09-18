package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

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
			// Assess("PaasNS creation with linked Paas", assertPaasNSCreated).
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

	// if err := cfg.Client().Resources().Create(ctx, paasNs); err != nil {
	// 	t.Fatal(err)
	// }

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

	// TODO: cleanup

	return ctx
}

// func assertPaasNSCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {

// }

func createPaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config, paasns api.PaasNS) api.PaasNS {

	if err := cfg.Client().Resources().Create(ctx, &paasns); err != nil {
		t.Fatalf("Failed to create PaasNS: %v", err)
	}

	return paasns
}

func getPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) (paas api.Paas, err error) {
	err = cfg.Client().Resources().Get(ctx, paasNsName, cfg.Namespace(), &paas)
	return paas, err
}

func getPaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config) api.PaasNS {
	var paasns api.PaasNS

	if err := cfg.Client().Resources().Get(ctx, paasNsName, cfg.Namespace(), &paasns); err != nil {
		t.Fatalf("Failed to retrieve PaasNS: %v", err)
	}

	return paasns
}

// func deletePaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config) api.PaasNS {
// 	var paasns api.PaasNS

// 	if err := cfg.Client().Resources().Delete(ctx, paasNsName, cfg.Namespace(), &paasns); err != nil {
// 		t.Fatalf("Failed to retrieve PaasNS: %v", err)
// 	}

// 	return paasns
// }

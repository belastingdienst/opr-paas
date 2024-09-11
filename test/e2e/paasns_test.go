package e2e

import (
	"context"
	"fmt"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const paasNsName = "a-paasns"
const paasName = "a-paas"

func TestPaasNS(t *testing.T) {
	testenv.Test(
		t,
		features.New("PaasNS").
			// Setup(createPaasFn(paasWithQuota, paasSpec)).
			Assess("shows correct status on creation without paas", assertPaasNSWithoutPaas).
			// Assess("is created", assertPaasNSCreated).
			Feature(),
	)
}

func assertPaasNSWithoutPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paasNs := &api.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paasNsName,
			Namespace: "paas-e2e",
		},
		Spec: api.PaasNSSpec{Paas: paasName}, // this paas does not exist but we do reference it
	}

	// create paasns including reference to non-existent paas
	if err := cfg.Client().Resources().Create(ctx, paasNs); err != nil {
		t.Fatal(err)
	}

	fetchedPaasNS := getPaasNS(ctx, t, cfg)

	// checking for vals...
	var errMsg = fetchedPaasNS.Status.Messages[0]
	fmt.Println(fetchedPaasNS.Name)
	fmt.Printf("messages:", fetchedPaasNS.Status.Messages)
	fmt.Println(errMsg)

	// TODO: test for error
	// assert.Equal(t, paasNsName, &paasns.Name)
	// assert.Contains(t, "cannot find PaaS", errMsg)

	return ctx
}

// func assertPaasNSCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {

// }

func getPaasNS(ctx context.Context, t *testing.T, cfg *envconf.Config) api.PaasNS {
	var paasns api.PaasNS

	if err := cfg.Client().Resources().Get(ctx, paasNsName, cfg.Namespace(), &paasns); err != nil {
		t.Fatalf("Failed to retrieve PaasNS: %v", err)
	}

	return paasns
}

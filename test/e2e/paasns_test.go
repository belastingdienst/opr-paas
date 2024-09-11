package e2e

import (
	"context"
	"fmt"
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
			Assess("paasns creation with reference to non-existing paas", assertPaasNSWithoutPaas).
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

	// fetch paas but expect it to error because it shouldn't have been created just because we referenced it
	_, errPaas := getPaas(ctx, t, cfg)

	// referenced paas still should not exist
	assert.Error(t, errPaas)

	waitForOperator()

	fetchedPaasNS := getPaasNS(ctx, t, cfg)

	// assert paasns status message
	fmt.Println("----------------")
	fmt.Println(fetchedPaasNS)
	fmt.Println(fetchedPaasNS.Name)
	fmt.Println(fetchedPaasNS.Status.Messages) // Error message disappeared but was there last week. Possibly a timing issue

	// checking for vals...
	var errMsg = fetchedPaasNS.Status.Messages[0]
	fmt.Println(errMsg)
	// assert.Equal(t, paasNsName, &paasns.Name)
	assert.Contains(t, errMsg, "cannot find PaaS")

	return ctx
}

// func assertPaasNSCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {

// }

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

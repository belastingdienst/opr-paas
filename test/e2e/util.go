package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	appv1 "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

// Duration to pause after Paas creation to wait for reconciliation.
const waitForOperatorDuration = 1 * time.Second

func waitForOperator() {
	time.Sleep(waitForOperatorDuration)
}

// createPaasFn accepts a Paas spec object and a name and creates the Paas resource.
func createPaasFn(name string, paasSpec api.PaasSpec) types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		paas := &api.Paas{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       paasSpec,
		}

		if err := cfg.Client().Resources().Create(ctx, paas); err != nil {
			t.Fatalf("Failed to create Paas resource: %v", err)
		}

		waitForOperator()

		return ctx
	}
}

// teardownPaasFn deletes the Paas if it still exists (e.g. if deleting the Paas is not part of the test steps, or if an
// earlier assertion failed causing the deletion step to be skipped).
// Can be called as `.Teardown(teardownPaasFn("paas-name"))`
func teardownPaasFn(paasName string) types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		paas := &api.Paas{ObjectMeta: metav1.ObjectMeta{Name: paasName}}

		if cfg.Client().Resources().Delete(ctx, paas) == nil {
			t.Logf("Paas %s deleted", paasName)
		}

		return ctx
	}
}

// getPaas retrieves the Paas with the associated name.
func getPaas(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) *api.Paas {
	paas := &api.Paas{}
	getOrFail(ctx, name, cfg.Namespace(), paas, t, cfg)

	return paas
}

// deletePaas deletes the Paas with the associated name.
func deletePaas(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) {
	paas := &api.Paas{ObjectMeta: metav1.ObjectMeta{Name: name}}

	if err := cfg.Client().Resources().Delete(ctx, paas); err != nil {
		t.Fatalf("Failed to delete Paas: %v", err)
	}

	waitForOperator()
}

func getOrFail[T k8s.Object](ctx context.Context, name string, namespace string, obj T, t *testing.T, cfg *envconf.Config) T {
	if err := cfg.Client().Resources().Get(ctx, name, namespace, obj); err != nil {
		t.Fatalf("Failed to get resource %s: %v", name, err)
	}

	return obj
}

func listOrFail[L k8s.ObjectList](ctx context.Context, namespace string, obj L, t *testing.T, cfg *envconf.Config) L {
	if err := cfg.Client().Resources(namespace).List(ctx, obj); err != nil {
		t.Fatalf("Failed to get resource list: %v", err)
	}

	return obj
}

func getNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config, name string) corev1.Namespace {
	var ns corev1.Namespace

	if err := cfg.Client().Resources().Get(ctx, name, cfg.Namespace(), &ns); err != nil {
		t.Fatalf("Failed to retrieve namespace: %v", err)
	}

	return ns
}

func getApplicationSet(ctx context.Context, t *testing.T, cfg *envconf.Config, applicationSetName string, namespace string) appv1.ApplicationSet {
	var applicationSet appv1.ApplicationSet

	if err := cfg.Client().Resources().Get(ctx, applicationSetName, namespace, &applicationSet); err != nil {
		t.Fatal(err)
	}

	return applicationSet
}

func getApplicationSetListEntries(applicationSet appv1.ApplicationSet) ([]string, error) {
	var jsonStrings []string

	for _, generator := range applicationSet.Spec.Generators {
		if generator.List != nil {
			for _, element := range generator.List.Elements {
				jsonStr, err := intArrayToString(element.Raw)
				if err != nil {
					return nil, fmt.Errorf("error converting int array to string: %w", err)
				}
				jsonStrings = append(jsonStrings, jsonStr)
			}
		}
	}

	return jsonStrings, nil
}

func intArrayToString(intArray []byte) (string, error) {
	byteSlice := make([]byte, len(intArray))
	for i, v := range intArray {
		byteSlice[i] = byte(v)
	}
	return string(byteSlice), nil
}

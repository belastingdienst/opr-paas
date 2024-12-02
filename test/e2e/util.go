package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	argo "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"

	apimachinerywait "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	// Interval for polling k8s to wait for resource changes
	waitInterval = 1 * time.Second
	waitTimeout  = 2 * time.Minute
)

// deleteResourceSync requests resource deletion and returns once k8s has successfully deleted it.
func deleteResourceSync(ctx context.Context, cfg *envconf.Config, obj k8s.Object) error {
	resources := cfg.Client().Resources()
	waitCond := conditions.New(resources).ResourceDeleted(obj)

	if err := resources.Delete(ctx, obj); err != nil {
		return fmt.Errorf("failed to delete resource %s: %w", obj.GetName(), err)
	}

	if err := waitForDefaultOpts(ctx, waitCond); err != nil {
		return fmt.Errorf("failed waiting for resource %s to be deleted: %w", obj.GetName(), err)
	}

	return nil
}

// waitForDefaultOpts calls `wait.For()` with a set of default options.
func waitForDefaultOpts(ctx context.Context, condition apimachinerywait.ConditionWithContextFunc) error {
	return wait.For(condition, wait.WithContext(ctx), wait.WithInterval(waitInterval), wait.WithTimeout(waitTimeout))
}

// getOrFail retrieves a resource from k8s, failing the test if there is an error.
func getOrFail[T k8s.Object](ctx context.Context, name string, namespace string, obj T, t *testing.T, cfg *envconf.Config) T {
	if err := cfg.Client().Resources().Get(ctx, name, namespace, obj); err != nil {
		t.Fatalf("Failed to get resource %s: %v", name, err)
	}

	return obj
}

// listOrFail retrieves a resource list from k8s, failing the test if there is an error.
func listOrFail[L k8s.ObjectList](ctx context.Context, namespace string, obj L, t *testing.T, cfg *envconf.Config) L {
	if err := cfg.Client().Resources(namespace).List(ctx, obj); err != nil {
		t.Fatalf("Failed to get resource list: %v", err)
	}

	return obj
}

// getApplicationSetListEntries returns the parsed elements of all list generators
// (https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-List/) present in the passed ApplicationSet.
func getApplicationSetListEntries(applicationSet *argo.ApplicationSet) ([]map[string]string, error) {
	entries := make([]map[string]string, 0)

	for _, generator := range applicationSet.Spec.Generators {
		if generator.List != nil {
			for _, element := range generator.List.Elements {
				parsed := map[string]string{}

				if err := json.Unmarshal(element.Raw, &parsed); err != nil {
					return nil, fmt.Errorf("error parsing elements as JSON: %w", err)
				}

				entries = append(entries, parsed)
			}
		}
	}

	return entries, nil
}

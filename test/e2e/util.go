package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	paasapi "github.com/belastingdienst/opr-paas/v3/api"
	"github.com/belastingdienst/opr-paas/v3/api/plugin"
	"github.com/belastingdienst/opr-paas/v3/pkg/fields"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerywait "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	// Interval for polling k8s to wait for resource changes
	waitInterval = 1 * time.Second
	waitTimeout  = 1 * time.Minute
)

var (
	forwardPort int
	pluginToken string
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
func getOrFail[T k8s.Object](
	ctx context.Context,
	name string,
	namespace string,
	obj T,
	t *testing.T,
	cfg *envconf.Config,
) T {
	if err := cfg.Client().Resources().Get(ctx, name, namespace, obj); err != nil {
		t.Fatalf("Failed to get resource %s: %v", name, err)
	}

	return obj
}

// getAndFail retrieves a resource from k8s, failing the test if it was successfully retrieved.
func failWhenExists[T k8s.Object](
	ctx context.Context,
	name string,
	namespace string,
	obj T,
	t *testing.T,
	cfg *envconf.Config,
) {
	if err := cfg.Client().Resources().Get(ctx, name, namespace, obj); err == nil {
		t.Fatalf("Resource %s should not be successfully retrieved", name)
	}
}

// listOrFail retrieves a resource list from k8s, failing the test if there is an error.
func listOrFail[L k8s.ObjectList](ctx context.Context, namespace string, obj L, t *testing.T, cfg *envconf.Config) L {
	if err := cfg.Client().Resources(namespace).List(ctx, obj); err != nil {
		t.Fatalf("Failed to get resource list: %v", err)
	}

	return obj
}

// getCapFieldsForPaas returns the parsed elements of all list generators
// (https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-List/)
// which are present in the passed ApplicationSet.
func getCapFieldsForPaas(port int, capName string, paasName string) (allEntries fields.ElementMap, err error) {
	url := fmt.Sprintf("http://localhost:%d/api/v1/getparams.execute", port)
	pluginRequest := plugin.Request{
		ApplicationSetName: capName,
		Input:              plugin.Input{Parameters: fields.ElementMap{"capability": capName}},
	}
	body, err := json.Marshal(pluginRequest)
	if err != nil {
		return nil, err
	}

	httpRequest, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", pluginToken))

	client := &http.Client{}
	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("server error: %s", httpResponse.Status)
	}

	var responseBody plugin.Response

	if err = json.NewDecoder(httpResponse.Body).Decode(&responseBody); err != nil {
		return nil, fmt.Errorf("cannot decode json: %w", err)
	}
	for _, customFields := range responseBody.Output.Parameters {
		if value, ok := customFields["paas"]; ok && value == paasName {
			return customFields, nil
		}
	}

	return nil, nil
}

// waitForStatus accepts a k8s object with a `.status.conditions` block, and waits until the resource has been updated
// and status conditions have been matched as per the passed function. Only conditions matching the current generation
// of the resource are passed to the match function. `oldGeneration` must contain the generation of the resource prior
// to its requested update. The `generation` of a resource only updates on changes to its spec.
// For new resources, use 0.
func waitForStatus(
	ctx context.Context,
	cfg *envconf.Config,
	obj paasapi.Resource,
	oldGeneration int64,
	match func(conds []metav1.Condition) bool,
) error {
	var fetched k8s.Object
	waitCond := conditions.New(cfg.Client().Resources()).
		ResourceMatch(obj, func(object k8s.Object) bool {
			fetched = object

			currentGen := object.GetGeneration()
			if currentGen <= oldGeneration {
				return false
			}

			// Filter out all non-current status conditions
			conds := make([]metav1.Condition, 0)
			for _, c := range *object.(paasapi.Resource).GetConditions() {
				if currentGen == c.ObservedGeneration {
					conds = append(conds, c)
				}
			}

			return match(conds)
		})

	if err := waitForDefaultOpts(ctx, waitCond); err != nil {
		return fmt.Errorf(
			"failed waiting for %s to be reconciled: %w and has status block: %v",
			fetched.GetName(),
			err,
			fetched.(paasapi.Resource).GetConditions(),
		)
	}

	return nil
}

// waitForCondition blocks until the given status condition is true.
func waitForCondition(
	ctx context.Context,
	cfg *envconf.Config,
	obj paasapi.Resource,
	oldGeneration int64,
	readyCondition string,
) error {
	return waitForStatus(ctx, cfg, obj, oldGeneration, func(conds []metav1.Condition) bool {
		return meta.IsStatusConditionTrue(conds, readyCondition)
	})
}

// createSync creates the resource, blocking until the given status condition is true.
func createSync(ctx context.Context, cfg *envconf.Config, obj paasapi.Resource, readyCondition string) error {
	if err := cfg.Client().Resources().Create(ctx, obj); err != nil {
		return fmt.Errorf("failed to create %s: %w", obj.GetName(), err)
	}

	return waitForCondition(ctx, cfg, obj, 0, readyCondition)
}

// updateSync updates the resource, blocking until the given status condition is true.
func updateSync(ctx context.Context, cfg *envconf.Config, obj paasapi.Resource, readyCondition string) error {
	gen := obj.GetGeneration()

	if err := cfg.Client().Resources().Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update %s: %w", obj.GetName(), err)
	}

	return waitForCondition(ctx, cfg, obj, gen, readyCondition)
}

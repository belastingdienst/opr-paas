package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	paasapi "github.com/belastingdienst/opr-paas/v4/api"
	"github.com/belastingdienst/opr-paas/v4/api/plugin"
	"github.com/belastingdienst/opr-paas/v4/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v4/pkg/fields"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	quotav1 "github.com/openshift/api/quota/v1"
	userv1 "github.com/openshift/api/user/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	apimachinerywait "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
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
	cliResources := cfg.Client().Resources()
	waitCond := conditions.New(cliResources).ResourceDeleted(obj)

	if err := cliResources.Delete(ctx, obj); err != nil {
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

func createPaasConfig(ctx context.Context, cfg *envconf.Config, paasConfig v1alpha2.PaasConfig) error {
	// Create PaasConfig resource for testing
	err := cfg.Client().Resources().Create(ctx, &paasConfig)
	if err != nil {
		return err
	}

	waitUntilPaasConfigExists := conditions.New(cfg.Client().Resources()).
		ResourceMatch(&paasConfig, func(obj k8s.Object) bool {
			return obj.(*v1alpha2.PaasConfig).Name == paasConfig.Name
		})
	return waitForDefaultOpts(ctx, waitUntilPaasConfigExists)
}

func retrieveBearerToken(
	ctx context.Context,
	cfg *envconf.Config,
	namespace string,
	secretName string,
	key string,
) error {
	var tokenSecret = &corev1.Secret{}
	if err := cfg.Client().Resources().Get(ctx, secretName, namespace, tokenSecret); err != nil {
		return err
	}
	token, ok := tokenSecret.Data[key]
	if !ok {
		return fmt.Errorf("%s not in data for secret %s.%s", key, namespace, secretName)
	}
	pluginToken = string(token)
	return nil
}

func registerSchemes(cfg *envconf.Config) error {
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return err
	}
	scheme := r.GetScheme()

	for _, install := range []func(*runtime.Scheme) error{
		v1alpha1.AddToScheme,
		v1alpha2.AddToScheme,
		quotav1.Install,
		userv1.Install,
	} {
		install(scheme)
	}

	return nil
}

func startPortForward(
	config *rest.Config,
	namespace string,
	serviceName string,
	port int,
) (int, func(), error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create clientset: %v", err)
	}
	podName, err := GetPodNameForService(clientset, namespace, serviceName)
	if err != nil {
		return 0, nil, err
	}
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return 0, nil, fmt.Errorf("error on round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	// 2. Setup communication channels
	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	// Buffer for error logs
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	// Port 0 makes OS select an open port
	ports := []string{fmt.Sprintf("0:%d", port)}

	pf, err := portforward.New(dialer, ports, stopChan, readyChan, out, errOut)
	if err != nil {
		return 0, nil, fmt.Errorf("error while creating portforwarder: %w", err)
	}

	// 3. Start the forwarder in a goroutine
	go func() {
		if fwdErr := pf.ForwardPorts(); fwdErr != nil {
			log.Fatalf("PortForward error: %v", fwdErr)
		}
	}()

	// 4. Wait for tunnel to be ready
	<-readyChan

	// 5. Read port
	forwardedPorts, err := pf.GetPorts()
	if err != nil {
		return 0, nil, fmt.Errorf("cannot retrieve port: %w", err)
	}
	localPort := int(forwardedPorts[0].Local)

	// Return port and cleanup func (so you can close the port forward)
	cleanup := func() {
		close(stopChan)
	}

	return localPort, cleanup, nil
}

// GetPodNameForService finds a pod belonging to a service
func GetPodNameForService(clientset *kubernetes.Clientset, namespace, serviceName string) (string, error) {
	// 1. get Service
	svc, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("cannot find service: %w", err)
	}

	// Check for selectors
	if len(svc.Spec.Selector) == 0 {
		return "", fmt.Errorf("service %s has no selector", serviceName)
	}

	// 2. change map to string
	set := labels.Set(svc.Spec.Selector)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}

	// 3. find pods
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return "", fmt.Errorf("error searching for pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods for this service %s", serviceName)
	}

	// 4. Use first pod that is in a running state
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return pod.Name, nil
		}
	}

	return "", fmt.Errorf("no running pods for this service %s", serviceName)
}

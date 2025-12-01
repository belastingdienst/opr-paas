/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"

	quotav1 "github.com/openshift/api/quota/v1"
	userv1 "github.com/openshift/api/user/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"

	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	systemNamespace      = "paas-system"
	generatorServiceName = "webhook-service"
	pluginPort           = 4355
)

var (
	testenv env.Environment

	examplePaasConfig = v1alpha2.PaasConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "paas-config",
		},
		Spec: v1alpha2.PaasConfigSpec{
			Capabilities: map[string]v1alpha2.ConfigCapability{
				"argocd": {
					AppSet: "argoas",
					DefaultPermissions: map[string][]string{
						"argo-service-argocd-application-controller": {"monitoring-edit"},
						"argo-service-applicationset-controller":     {"monitoring-edit"},
					},
					ExtraPermissions: map[string][]string{
						"argo-service-argocd-application-controller": {"admin"},
					},
					QuotaSettings: v1alpha2.ConfigQuotaSettings{
						DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
							corev1.ResourceLimitsCPU:       resourcev1.MustParse("5"),
							corev1.ResourceLimitsMemory:    resourcev1.MustParse("4Gi"),
							corev1.ResourceRequestsCPU:     resourcev1.MustParse("1"),
							corev1.ResourceRequestsMemory:  resourcev1.MustParse("1Gi"),
							corev1.ResourceRequestsStorage: resourcev1.MustParse("0"),
							// revive:disable-next-line
							corev1.ResourceName("thin.storageclass.storage.k8s.io/persistentvolumeclaims"): resourcev1.MustParse(
								"0",
							),
						},
					},
					CustomFields: map[string]v1alpha2.ConfigCustomField{
						"git_url": {
							Required: true,
							// in yaml you need escaped slashes: '^ssh:\/\/git@scm\/[a-zA-Z0-9-.\/]*.git$'
							Validation: "^ssh://git@scm/[a-zA-Z0-9-./]*.git$",
						},
						"git_revision": {
							Default: "main",
						},
						"git_path": {
							Default: ".",
							// in yaml you need escaped slashes: '^[a-zA-Z0-9.\/]*$'
							Validation: "^[a-zA-Z0-9./]*$",
						},
					},
				},
				"cap5": {
					AppSet: "cap5as",
					QuotaSettings: v1alpha2.ConfigQuotaSettings{
						DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
							corev1.ResourceLimitsCPU:       resourcev1.MustParse("6"),
							corev1.ResourceLimitsMemory:    resourcev1.MustParse("7Gi"),
							corev1.ResourceRequestsCPU:     resourcev1.MustParse("5"),
							corev1.ResourceRequestsMemory:  resourcev1.MustParse("6Gi"),
							corev1.ResourceRequestsStorage: resourcev1.MustParse("0"),
							// revive:disable-next-line
							corev1.ResourceName("thin.storageclass.storage.k8s.io/persistentvolumeclaims"): resourcev1.MustParse(
								"0",
							),
						},
					},
				},
				"tekton": {
					AppSet: "tektonas",
					DefaultPermissions: map[string][]string{
						"pipeline": {"view", "alert-routing-edit"},
					},
					ExtraPermissions: map[string][]string{
						"pipeline": {"admin"},
					},
					QuotaSettings: v1alpha2.ConfigQuotaSettings{
						Clusterwide: true,
						DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
							corev1.ResourceLimitsCPU:       resourcev1.MustParse("5"),
							corev1.ResourceLimitsMemory:    resourcev1.MustParse("8Gi"),
							corev1.ResourceRequestsCPU:     resourcev1.MustParse("1"),
							corev1.ResourceRequestsMemory:  resourcev1.MustParse("2Gi"),
							corev1.ResourceRequestsStorage: resourcev1.MustParse("100Gi"),
							// revive:disable-next-line
							corev1.ResourceName("thin.storageclass.storage.k8s.io/persistentvolumeclaims"): resourcev1.MustParse(
								"0",
							),
						},
						MinQuotas: map[corev1.ResourceName]resourcev1.Quantity{
							corev1.ResourceLimitsCPU:    resourcev1.MustParse("5"),
							corev1.ResourceLimitsMemory: resourcev1.MustParse("4Gi"),
						},
						MaxQuotas: map[corev1.ResourceName]resourcev1.Quantity{
							corev1.ResourceLimitsCPU:    resourcev1.MustParse("10"),
							corev1.ResourceLimitsMemory: resourcev1.MustParse("10Gi"),
						},
						Ratio: 0.1,
					},
				},
				"sso": {
					AppSet: "ssoas",
					QuotaSettings: v1alpha2.ConfigQuotaSettings{
						Clusterwide: false,
						DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
							corev1.ResourceLimitsCPU:       resourcev1.MustParse("1"),
							corev1.ResourceLimitsMemory:    resourcev1.MustParse("512Mi"),
							corev1.ResourceRequestsCPU:     resourcev1.MustParse("100m"),
							corev1.ResourceRequestsMemory:  resourcev1.MustParse("128Mi"),
							corev1.ResourceRequestsStorage: resourcev1.MustParse("0"),
							// revive:disable-next-line
							corev1.ResourceName("thin.storageclass.storage.k8s.io/persistentvolumeclaims"): resourcev1.MustParse(
								"0",
							),
						},
					},
				},
				"grafana": {
					AppSet: "grafanaas",
					QuotaSettings: v1alpha2.ConfigQuotaSettings{
						DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
							corev1.ResourceLimitsCPU:       resourcev1.MustParse("2"),
							corev1.ResourceLimitsMemory:    resourcev1.MustParse("2Gi"),
							corev1.ResourceRequestsCPU:     resourcev1.MustParse("500m"),
							corev1.ResourceRequestsMemory:  resourcev1.MustParse("512Mi"),
							corev1.ResourceRequestsStorage: resourcev1.MustParse("2Gi"),
							// revive:disable-next-line
							corev1.ResourceName("thin.storageclass.storage.k8s.io/persistentvolumeclaims"): resourcev1.MustParse(
								"0",
							),
						},
					},
				},
				"capexternal": {
					QuotaSettings: v1alpha2.ConfigQuotaSettings{DefQuota: nil, MinQuotas: nil, MaxQuotas: nil},
				},
			},
			ClusterWideArgoCDNamespace: "asns",
			Debug:                      false,
			DecryptKeysSecret: v1alpha2.NamespacedName{
				Name:      "example-keys",
				Namespace: "paas-system",
			},
			ManagedByLabel:  "argocd.argoproj.io/manby",
			ManagedBySuffix: "argocd",
			RequestorLabel:  "o.lbl",
			QuotaLabel:      "q.lbl",
			RoleMappings: map[string][]string{
				"default": {"admin"},
				"viewer":  {"view"},
			},
			Templating: v1alpha2.ConfigTemplatingItems{
				GenericCapabilityFields: v1alpha2.ConfigTemplatingItem{
					"requestor":  "{{ .Paas.Spec.Requestor }}",
					"service":    "{{ (split \"-\" .Paas.Name)._0 }}",
					"subservice": "{{ (split \"-\" .Paas.Name)._1 }}",
				},
			},
		},
	}
)

// end examplePaasConfig

func createPaasConfig(ctx context.Context, cfg *envconf.Config) error {
	paasconfig := &v1alpha2.PaasConfig{}
	*paasconfig = examplePaasConfig

	// Create PaasConfig resource for testing
	err := cfg.Client().Resources().Create(ctx, paasconfig)
	if err != nil {
		return err
	}

	waitUntilPaasConfigExists := conditions.New(cfg.Client().Resources()).
		ResourceMatch(paasconfig, func(obj k8s.Object) bool {
			return obj.(*v1alpha2.PaasConfig).Name == paasconfig.Name
		})
	return waitForDefaultOpts(ctx, waitUntilPaasConfigExists)
}

func retrieveBearerToken(ctx context.Context, cfg *envconf.Config) error {
	var tokenSecret = &corev1.Secret{}
	if err := cfg.Client().Resources().Get(ctx, "generator-token", systemNamespace, tokenSecret); err != nil {
		return err
	}
	token, ok := tokenSecret.Data["ARGOCD_GENERATOR_TOKEN"]
	if !ok {
		return errors.New("ARGOCD_GENERATOR_TOKEN not in generator token data")
	}
	pluginToken = string(token)
	return nil
}

func TestMain(m *testing.M) {
	testenv = env.New()

	// ResolveKubeConfigFile() function is called to get kubeconfig loaded,
	// it uses either `--kubeconfig` flag, `KUBECONFIG` env or by default ` $HOME/.kube/config` path.
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	testenv = env.NewWithConfig(cfg)
	e2eNamespace := "paas-e2e"

	if envNamespace := os.Getenv("PAAS_E2E_NS"); envNamespace != "" {
		e2eNamespace = envNamespace
		cfg = cfg.WithNamespace(e2eNamespace)
	} else {
		testenv.Setup(
			envfuncs.CreateNamespace(e2eNamespace),
		)
		testenv.Finish(
			envfuncs.DeleteNamespace(e2eNamespace),
		)
	}

	var pfDone func()
	// Global setup
	testenv.Setup(
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			err := createPaasConfig(ctx, cfg)
			if err != nil {
				return ctx, err
			}
			err = retrieveBearerToken(ctx, cfg)
			if err != nil {
				return ctx, err
			}
			forwardPort, pfDone, err = startPortForward(cfg.Client().RESTConfig())
			if err != nil {
				return ctx, err
			}

			return ctx, nil
		})

	// Global teardown
	testenv.Finish(
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			// Delete the PaasConfig resource
			paasConfig := &v1alpha2.PaasConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "paas-config",
				},
			}

			err := deleteResourceSync(ctx, cfg, paasConfig)
			if err != nil {
				return ctx, err
			}
			pfDone()

			return ctx, nil
		},
	)

	if err := registerSchemes(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register schemes: %v", err)
		os.Exit(1)
	}

	// Run tests
	os.Exit(testenv.Run(m))
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
) (int, func(), error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return 0, nil, fmt.Errorf("Failed to create clientset: %v", err)
	}
	podName, err := GetPodNameForService(clientset, systemNamespace, generatorServiceName)
	if err != nil {
		return 0, nil, err
	}
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(systemNamespace).
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
	ports := []string{fmt.Sprintf("0:%d", pluginPort)}

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

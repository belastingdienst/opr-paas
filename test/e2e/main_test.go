/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"

	argoprojv1alpha1 "github.com/belastingdienst/opr-paas/v2/internal/stubs/argoproj/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/belastingdienst/opr-paas/v2/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/v2/api/v1alpha2"

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

var testenv env.Environment

var examplePaasConfig = v1alpha2.PaasConfig{
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
		ResourceLabels: v1alpha2.ConfigResourceLabelConfigs{
			AppSetLabels: v1alpha2.ConfigResourceLabelConfig{
				"requestor":  "{{ .Paas.Spec.Requestor }}",
				"service":    "{{ (split \"-\" .Paas.Name)._0 }}",
				"subservice": "{{ (split \"-\" .Paas.Name)._1 }}",
			},
		},
	},
}

// end examplePaasConfig

func TestMain(m *testing.M) {
	testenv = env.New()

	// ResolveKubeConfigFile() function is called to get kubeconfig loaded,
	// it uses either `--kubeconfig` flag, `KUBECONFIG` env or by default ` $HOME/.kube/config` path.
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	testenv = env.NewWithConfig(cfg)
	namespace := "paas-e2e"

	if envNamespace := os.Getenv("PAAS_E2E_NS"); envNamespace != "" {
		namespace = envNamespace
		cfg = cfg.WithNamespace(namespace)
	} else {
		testenv.Setup(
			envfuncs.CreateNamespace(namespace),
		)
		testenv.Finish(
			envfuncs.DeleteNamespace(namespace),
		)
	}

	// Global setup
	testenv.Setup(
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			paasconfig := &v1alpha2.PaasConfig{}
			*paasconfig = examplePaasConfig

			// Create PaasConfig resource for testing
			err := cfg.Client().Resources().Create(ctx, paasconfig)
			if err != nil {
				return ctx, err
			}

			waitUntilPaasConfigExists := conditions.New(cfg.Client().Resources()).
				ResourceMatch(paasconfig, func(obj k8s.Object) bool {
					return obj.(*v1alpha2.PaasConfig).Name == paasconfig.Name
				})

			if err := waitForDefaultOpts(ctx, waitUntilPaasConfigExists); err != nil {
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
		argoprojv1alpha1.AddToScheme,
	} {
		install(scheme)
	}

	return nil
}

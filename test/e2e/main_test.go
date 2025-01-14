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

	argoprojlabsv1beta1 "github.com/belastingdienst/opr-paas/internal/stubs/argoproj-labs/v1beta1"
	argoprojv1alpha1 "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

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

var examplePaasConfig = v1alpha1.PaasConfig{
	ObjectMeta: metav1.ObjectMeta{
		Name: "paas-config",
	},
	Spec: v1alpha1.PaasConfigSpec{
		ClusterWideArgoCDNamespace: "asns",
		ArgoPermissions: v1alpha1.ConfigArgoPermissions{
			ResourceName:  "argocd",
			DefaultPolicy: "role:tester",
			Role:          "admin",
			Header:        "g, system:cluster-admins, role:admin",
		},
		Capabilities: map[string]v1alpha1.ConfigCapability{
			"argocd": {
				AppSet: "argoas",
				DefaultPermissions: map[string][]string{
					"argo-service-argocd-application-controller": {"monitoring-edit"},
					"argo-service-applicationset-controller":     {"monitoring-edit"},
				},
				ExtraPermissions: map[string][]string{
					"argo-service-argocd-application-controller": {"admin"},
				},
				QuotaSettings: v1alpha1.ConfigQuotaSettings{
					DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
						corev1.ResourceLimitsCPU:       resource.MustParse("5"),
						corev1.ResourceLimitsMemory:    resource.MustParse("4Gi"),
						corev1.ResourceRequestsCPU:     resource.MustParse("1"),
						corev1.ResourceRequestsMemory:  resource.MustParse("1Gi"),
						corev1.ResourceRequestsStorage: resource.MustParse("0"),
						corev1.ResourceName("thin.storageclass.storage.k8s.io/persistentvolumeclaims"): resource.MustParse("0"),
					},
				},
				CustomFields: map[string]v1alpha1.ConfigCustomField{
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
				QuotaSettings: v1alpha1.ConfigQuotaSettings{
					DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
						corev1.ResourceLimitsCPU:       resource.MustParse("6"),
						corev1.ResourceLimitsMemory:    resource.MustParse("7Gi"),
						corev1.ResourceRequestsCPU:     resource.MustParse("5"),
						corev1.ResourceRequestsMemory:  resource.MustParse("6Gi"),
						corev1.ResourceRequestsStorage: resource.MustParse("0"),
						corev1.ResourceName("thin.storageclass.storage.k8s.io/persistentvolumeclaims"): resource.MustParse("0"),
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
				QuotaSettings: v1alpha1.ConfigQuotaSettings{
					Clusterwide: true,
					DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
						corev1.ResourceLimitsCPU:       resource.MustParse("5"),
						corev1.ResourceLimitsMemory:    resource.MustParse("8Gi"),
						corev1.ResourceRequestsCPU:     resource.MustParse("1"),
						corev1.ResourceRequestsMemory:  resource.MustParse("2Gi"),
						corev1.ResourceRequestsStorage: resource.MustParse("100Gi"),
						corev1.ResourceName("thin.storageclass.storage.k8s.io/persistentvolumeclaims"): resource.MustParse("0"),
					},
					MinQuotas: map[corev1.ResourceName]resourcev1.Quantity{
						corev1.ResourceLimitsCPU:    resource.MustParse("5"),
						corev1.ResourceLimitsMemory: resource.MustParse("4Gi"),
					},
					MaxQuotas: map[corev1.ResourceName]resourcev1.Quantity{
						corev1.ResourceLimitsCPU:    resource.MustParse("1"),
						corev1.ResourceLimitsMemory: resource.MustParse("1Gi"),
					},
					Ratio: 10,
				},
			},
			"sso": {
				AppSet: "ssoas",
				QuotaSettings: v1alpha1.ConfigQuotaSettings{
					Clusterwide: false,
					DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
						corev1.ResourceLimitsCPU:       resource.MustParse("1"),
						corev1.ResourceLimitsMemory:    resource.MustParse("512Mi"),
						corev1.ResourceRequestsCPU:     resource.MustParse("100m"),
						corev1.ResourceRequestsMemory:  resource.MustParse("128Mi"),
						corev1.ResourceRequestsStorage: resource.MustParse("0"),
						corev1.ResourceName("thin.storageclass.storage.k8s.io/persistentvolumeclaims"): resource.MustParse("0"),
					},
				},
			},
			"grafana": {
				AppSet: "grafanaas",
				QuotaSettings: v1alpha1.ConfigQuotaSettings{
					DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
						corev1.ResourceLimitsCPU:       resource.MustParse("2"),
						corev1.ResourceLimitsMemory:    resource.MustParse("2Gi"),
						corev1.ResourceRequestsCPU:     resource.MustParse("500m"),
						corev1.ResourceRequestsMemory:  resource.MustParse("512Mi"),
						corev1.ResourceRequestsStorage: resource.MustParse("2Gi"),
						corev1.ResourceName("thin.storageclass.storage.k8s.io/persistentvolumeclaims"): resource.MustParse("0"),
					},
				},
			},
		},
		Debug:           false,
		DecryptKeyPaths: []string{"/tmp/paas-e2e/secrets/priv"},
		LDAP: v1alpha1.ConfigLdap{
			Host: "my-ldap-host",
			Port: 13,
		},
		ManagedByLabel: "argocd.argoproj.io/manby",
		RequestorLabel: "o.lbl",
		QuotaLabel:     "q.lbl",
		RoleMappings: map[string][]string{
			"default": {"admin"},
			"viewer":  {"view"},
		},
		GroupSyncList: v1alpha1.NamespacedName{
			Namespace: "gsns",
			Name:      "wlname",
		},
		GroupSyncListKey:  "groupsynclist.txt",
		ExcludeAppSetName: "whatever",
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
			paasconfig := &v1alpha1.PaasConfig{}
			*paasconfig = examplePaasConfig

			// Create PaasConfig resource for testing
			err := cfg.Client().Resources().Create(ctx, paasconfig)
			if err != nil {
				return ctx, err
			}

			waitUntilPaasConfigExists := conditions.New(cfg.Client().Resources()).ResourceMatch(paasconfig, func(obj k8s.Object) bool {
				return obj.(*v1alpha1.PaasConfig).Name == paasconfig.Name
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
			paasConfig := &v1alpha1.PaasConfig{
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

	if err = v1alpha1.AddToScheme(scheme); err != nil {
		return err
	} else if err = quotav1.AddToScheme(scheme); err != nil {
		return err
	} else if err = userv1.AddToScheme(scheme); err != nil {
		return err
	} else if err = argoprojv1alpha1.AddToScheme(scheme); err != nil {
		return err
	} else if err = argoprojlabsv1beta1.AddToScheme(scheme); err != nil {
		return err
	}

	return nil
}

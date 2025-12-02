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

	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"

	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	systemNamespace      = "paas-system"
	tokenSecretName      = "generator-token"
	tokenSecretKey       = "ARGOCD_GENERATOR_TOKEN"
	generatorServiceName = "webhook-service"
	generatorServicePort = 4355
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
			err := createPaasConfig(ctx, cfg, examplePaasConfig)
			if err != nil {
				return ctx, err
			}
			err = retrieveBearerToken(ctx, cfg, systemNamespace, tokenSecretName, tokenSecretKey)
			if err != nil {
				return ctx, err
			}
			forwardPort, pfDone, err = startPortForward(
				cfg.Client().RESTConfig(),
				systemNamespace,
				generatorServiceName,
				generatorServicePort,
			)
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

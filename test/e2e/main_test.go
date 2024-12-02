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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	api "github.com/belastingdienst/opr-paas/api/v1alpha1"

	quotav1 "github.com/openshift/api/quota/v1"
	userv1 "github.com/openshift/api/user/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/utils"

	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var testenv env.Environment

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

	testenv.Setup(
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			// install PaasConfig instance
			if p := utils.RunCommand(
				fmt.Sprintf("kubectl apply -f %s", "../../manifests/config/example-paasconfig.yaml"),
			); p.Err() != nil {
				return ctx, p.Err()
			}

			paasconfig := &v1alpha1.PaasConfig{
				ObjectMeta: v1.ObjectMeta{
					Name: "paas-config",
				},
			}
			waitUntilPaasConfigExists := conditions.New(cfg.Client().Resources()).ResourceMatch(paasconfig, func(obj k8s.Object) bool {
				return obj.(*api.PaasConfig).Name == paasconfig.Name
			})

			if err := waitForDefaultOpts(ctx, waitUntilPaasConfigExists); err != nil {
				return nil, err
			}

			return ctx, nil
		})

	if err := registerSchemes(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register schemes: %v", err)
		os.Exit(1)
	}

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

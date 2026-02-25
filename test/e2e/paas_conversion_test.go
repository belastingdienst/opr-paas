package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/v4/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

const (
	paasv2Name = "paas-v1alpha2"
	// revive:disable-next-line
	paasV1alpha2Secret = "M9rkiqfVqvE5kjMkaZLt8jokIIAuVLfTS8dXFQa3drmOyIFWSzHJym1PKyzkwnK07vcJkxbfEkO22IbpkziXxrF1OflpNMzIcFFALMw472sczeeJDPvl1u6/F14agq4avc/Osk0zreRLRPS2jkhXE8VnbNsi+//PuRssCbp/ink8mpMg7mVKL9BfQXBu37KppvXEfOA+M6C4ZkNIVqrl7HcRW/e296GpCFkbQ7qa6JWwmgR22j64hcFJDorWhALAuGj7lZ/Wsm0ZzuFFD9tRKuFnxMFRlfDPMm26+NyXTUPNEZuqfeswaa8TLv/ldjr4Y78e+F3q5G0IGFj2sdTp08SMkLDfa8eYfxqa83EWQjiJcxggrPUs2eZZ0hN/IjxDjRh/nwSrKfugk/SQL61jC7slB8Beh8xurfpw/YEOwwooItkjp+1kliDLepUgixm9iY6Mrk4oNfOl2Ul2xggnijd4q2mQ8sPXf++R7ntV5zdcvKW411b93d9CTLgf+I2+2dqYK2TqPZmzVOPqigx1bIGCpbsD6xQH/QcuOPSOnvluDTJKFx3jENwzQ41wXr06Uv45WIUcdgUKwQkRFJ/dBeaQyBiB+oBXu3PcTsXi6MziHbPdxH0Xnv1SPkVnd0oFbxqwzhtabvKnc/opuaTosaDCWMjdRoJFh01rs4MdELQ="
)

func createAlpha1PaasWithCondFn(name string, paasSpec api.PaasSpec, readyCondition string) types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		paas := &api.Paas{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       paasSpec,
		}

		if err := createSync(ctx, cfg, paas, readyCondition); err != nil {
			t.Fatal(err)
		}

		return ctx
	}
}

func TestPaasConversion(t *testing.T) {
	v1Spec := api.PaasSpec{
		Requestor: "paas-user",
		Quota: map[corev1.ResourceName]resource.Quantity{
			"cpu":    resource.MustParse("200m"),
			"memory": resource.MustParse("256Mi"),
		},
		Capabilities: api.PaasCapabilities{
			"argocd": {
				Enabled:     true,
				GitURL:      "ssh://git@scm/repo.git",
				GitRevision: "main",
				CustomFields: map[string]string{
					"git_path": ".",
				},
			},
			"sso": {Enabled: false},
			"tekton": {
				Enabled: true,
				SSHSecrets: map[string]string{
					paasArgoGitURL: paasArgoSecret,
				},
			},
		},
		Namespaces: []string{"foo", "bar"},
	}

	testenv.Test(
		t,
		features.New("Conversion between Paas versions").
			Setup(createAlpha1PaasWithCondFn(paasWithArgo, v1Spec, api.TypeReadyPaas)).
			Assess("converted to v1alpha2 when requested", assertV2Conversion).
			Assess("v1alpha2 can be created", assertV2Created).
			Assess("v1alpha2 retrieved as v1alpha1", assertV1Conversion).
			Teardown(teardownPaasFn(paasWithArgo)).
			Teardown(teardownPaasFn(paasv2Name)).
			Feature(),
	)
}

func assertV2Conversion(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := v1alpha2.Paas{}
	require.NoError(t, cfg.Client().Resources().Get(ctx, paasWithArgo, cfg.Namespace(), &paas))

	assert.Len(t, paas.Spec.Capabilities, 2)
	// SSO shouldn't be converted as it was not Enabled in v1alpha1
	_, exists := paas.Spec.Capabilities["sso"]
	assert.False(t, exists)
	assert.Equal(
		t,
		map[string]string{
			"git_url":      "ssh://git@scm/repo.git",
			"git_revision": "main",
			"git_path":     ".",
		},
		paas.Spec.Capabilities["argocd"].CustomFields,
	)
	assert.Equal(
		t,
		map[string]string{
			paasArgoGitURL: paasArgoSecret,
		},
		paas.Spec.Secrets,
	)

	assert.Len(t, paas.Spec.Namespaces, 2)
	assert.Contains(t, paas.Spec.Namespaces, "foo")
	assert.Contains(t, paas.Spec.Namespaces, "bar")

	return ctx
}

func assertV2Created(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paasv2 := v1alpha2.Paas{
		ObjectMeta: metav1.ObjectMeta{Name: paasv2Name},
		Spec: v1alpha2.PaasSpec{
			Requestor: "paas-user",
			Quota: map[corev1.ResourceName]resource.Quantity{
				"cpu":    resource.MustParse("200m"),
				"memory": resource.MustParse("256Mi"),
			},
			Capabilities: v1alpha2.PaasCapabilities{
				"argocd": {
					CustomFields: map[string]string{
						"git_url":      "ssh://git@scm/repo.git",
						"git_revision": "main",
						"git_path":     ".",
					},
				},
				"sso": {},
				"tekton": {
					Secrets: map[string]string{
						paasArgoGitURL: paasV1alpha2Secret,
					},
				},
			},
			Namespaces: v1alpha2.PaasNamespaces{
				"foo": {},
				"bar": {
					Secrets: map[string]string{
						"foo": paasV1alpha2Secret,
					},
				},
			},
		},
	}

	assert.NoError(t, createSync(ctx, cfg, &paasv2, v1alpha2.TypeReadyPaas))

	return ctx
}

func assertV1Conversion(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := api.Paas{}
	require.NoError(t, cfg.Client().Resources().Get(ctx, paasv2Name, cfg.Namespace(), &paas))

	assert.Len(t, paas.Spec.Capabilities, 3)
	assert.Len(t, paas.Spec.Capabilities["argocd"].CustomFields, 3)
	assert.Equal(t, "ssh://git@scm/repo.git", paas.Spec.Capabilities["argocd"].GitURL)
	assert.Equal(t, "main", paas.Spec.Capabilities["argocd"].GitRevision)
	assert.Equal(t, ".", paas.Spec.Capabilities["argocd"].GitPath)
	assert.Equal(
		t,
		map[string]string{
			paasArgoGitURL: paasV1alpha2Secret,
		},
		paas.Spec.Capabilities["tekton"].SSHSecrets,
	)

	assert.Equal(t, []string{"bar", "foo"}, paas.Spec.Namespaces)

	return ctx
}

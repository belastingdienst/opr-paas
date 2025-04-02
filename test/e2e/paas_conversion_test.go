package e2e

import (
	"context"
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/internal/quota"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasv1Name = "paas-v1alpha1"
	paasv2Name = "paas-v1alpha2"
)

func TestPaasConversion(t *testing.T) {
	v1Spec := v1alpha1.PaasSpec{
		Requestor: "paas-user",
		Quota:     make(quota.Quota),
		Capabilities: v1alpha1.PaasCapabilities{
			"argocd": {
				Enabled:     true,
				GitURL:      "ssh://git@scm/repo.git",
				GitRevision: "main",
				CustomFields: map[string]string{
					"git_path": ".",
				},
			},
			"sso": {Enabled: true},
			"tekton": {
				Enabled: false,
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
			Setup(createPaasFn(paasv1Name, v1Spec)).
			Assess("converted to v1alpha2 when requested", assertV2Conversion).
			Assess("v1alpha2 can be created", assertV2Created).
			Assess("v1alpha2 retrieved as v1alpha1", assertV1Conversion).
			Teardown(teardownPaasFn(paasv1Name)).
			Teardown(teardownPaasFn(paasv2Name)).
			Feature(),
	)
}

func assertV2Conversion(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := v1alpha2.Paas{}
	require.NoError(t, cfg.Client().Resources().Get(ctx, paasv1Name, cfg.Namespace(), &paas))

	assert.Len(t, paas.Spec.Capabilities, 3)
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
		paas.Spec.Capabilities["tekton"].Secrets,
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
			Quota:     make(quota.Quota),
			Capabilities: v1alpha2.PaasCapabilities{
				"argocd": {
					CustomFields: map[string]string{
						"git_url":      "ssh://git@scm/repo.git",
						"git_revision": "main",
					},
				},
				"sso": {},
				"tekton": {
					Secrets: map[string]string{
						paasArgoGitURL: paasArgoSecret,
					},
				},
			},
			Namespaces: v1alpha2.PaasNamespaces{
				"foo": {},
				"bar": {
					Secrets: map[string]string{
						"foo": "bar",
					},
				},
			},
		},
	}

	assert.NoError(t, createSync(ctx, cfg, &paasv2, v1alpha2.TypeReadyPaas))

	return ctx
}

func assertV1Conversion(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := v1alpha1.Paas{}
	require.NoError(t, cfg.Client().Resources().Get(ctx, paasv2Name, cfg.Namespace(), &paas))

	assert.Len(t, paas.Spec.Capabilities, 3)
	assert.Len(t, paas.Spec.Capabilities["argocd"].CustomFields, 0)
	assert.Equal(t, "ssh://git@scm/repo.git", paas.Spec.Capabilities["argocd"].GitURL)
	assert.Equal(t, "main", paas.Spec.Capabilities["argocd"].GitRevision)
	assert.Equal(
		t,
		map[string]string{
			paasArgoGitURL: paasArgoSecret,
		},
		paas.Spec.Capabilities["tekton"].SSHSecrets,
	)

	assert.Equal(t, []string{"bar", "foo"}, paas.Spec.Namespaces)

	return ctx
}

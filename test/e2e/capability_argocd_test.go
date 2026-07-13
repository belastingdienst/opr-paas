package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v5/pkg/fields"
	"github.com/belastingdienst/opr-paas/v5/pkg/quota"

	quotav1 "github.com/openshift/api/quota/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	argoCapName         = "argocd"
	paasWithArgo        = "paas-capability-argocd"
	paasArgoNs          = paasWithArgo + "-argocd"
	paasArgoGitURL      = "ssh://git@scm/some-repo.git"
	paasArgoGitPath     = "foo/"
	paasArgoGitRevision = "main"
	paasRequestor       = "paas-requestor"
)

func TestCapabilityArgoCD(t *testing.T) {
	paasArgoSecret := encryptE2ESecret(t, paasWithArgo, "dummysecret")

	paasSpec := api.PaasSpec{
		Requestor: paasRequestor,
		Quota: quota.Quota{
			corev1.ResourceLimitsCPU: resource.MustParse("5"),
		},
		Capabilities: api.PaasCapabilities{
			argoCapName: api.PaasCapability{
				CustomFields: map[string]string{
					"git_path":     paasArgoGitPath,
					"git_revision": paasArgoGitRevision,
					"git_url":      paasArgoGitURL,
				},
				Secrets:          map[string]string{paasArgoGitURL: paasArgoSecret},
				ExtraPermissions: true,
			},
		},
	}

	testenv.Test(
		t,
		features.New("ArgoCD Capability").
			Setup(createPaasFn(paasWithArgo, paasSpec)).
			Assess("Argo cap is created", assertArgoCapCreated).
			Assess("Argo cap is updated", assertArgoCapUpdated).
			Assess("Argo cap has ClusterRoleBindings", assertArgoCRB).
			Teardown(teardownPaasFn(paasWithArgo)).
			Feature(),
	)
}

func assertArgoCapCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	entries, err := getCapFieldsForPaas(forwardPort, argoCapName, paasWithArgo)
	require.NoError(t, err)

	assert.Equal(t, fields.ElementMap{
		"git_path":     paasArgoGitPath,
		"git_revision": paasArgoGitRevision,
		"git_url":      paasArgoGitURL,
		// when no groups are set, field is left out
		"paas":       paasWithArgo,
		"requestor":  paasRequestor,
		"service":    "paas",
		"subservice": "capability",
	}, entries)

	assert.NotNil(
		t,
		getOrFail(ctx, paasArgoNs, corev1.NamespaceAll, &corev1.Namespace{}, t, cfg),
		"ArgoCD namespace created",
	)

	secrets := listOrFail(ctx, paasArgoNs, &corev1.SecretList{}, t, cfg).Items
	assert.Len(t, secrets, 1)
	assert.Equal(
		t,
		"dummysecret",
		string(secrets[0].Data["sshPrivateKey"]),
		"SSH secret is created in ArgoCD namespace",
	)

	crq := getOrFail(ctx, paasArgoNs, corev1.NamespaceAll, &quotav1.ClusterResourceQuota{}, t, cfg)
	assert.Equal(
		t,
		"q.lbl="+paasArgoNs,
		metav1.FormatLabelSelector(crq.Spec.Selector.LabelSelector),
		"Quota selects ArgoCD namespace via selector set to `quota_label` configuration",
	)
	assert.Equal(t, corev1.ResourceList{
		corev1.ResourceLimitsCPU:       resource.MustParse("5"),
		corev1.ResourceRequestsCPU:     resource.MustParse("1"),
		corev1.ResourceLimitsMemory:    resource.MustParse("4Gi"),
		corev1.ResourceRequestsMemory:  resource.MustParse("1Gi"),
		corev1.ResourceRequestsStorage: resource.MustParse("0"),
	}, crq.Spec.Quota.Hard, "Quota conforms to defaults from Paas config")

	return ctx
}

func assertArgoCapUpdated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	updatedRevision := "updatedRevision"
	paas := getPaas(ctx, paasWithArgo, t, cfg)
	paasArgoSecret := paas.Spec.Capabilities[argoCapName].Secrets[paasArgoGitURL]

	paas.Spec.Capabilities = api.PaasCapabilities{
		argoCapName: api.PaasCapability{
			Secrets:          map[string]string{paasArgoGitURL: paasArgoSecret},
			ExtraPermissions: true,
			CustomFields: map[string]string{
				"git_path":     paasArgoGitPath,
				"git_revision": updatedRevision,
				"git_url":      paasArgoGitURL,
			},
		},
	}
	paas.Spec.Groups = api.PaasGroups{
		"mygroup": api.PaasGroup{
			Users: []string{"user1"},
			Roles: []string{"viewer"},
		},
	}

	// As only the Paas spec is updated via the above change, we wait for that and
	// know that no reconciliation of PaasNs takes place so no need to wait for that.
	// check #185 for more details
	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}
	entries, err := getCapFieldsForPaas(forwardPort, argoCapName, paasWithArgo)
	require.NoError(t, err)

	// For now this still applies, later we move the git_.. properties to the appSet as well
	// Assert AppSet entry updated accordingly
	assert.Equal(t, entries,
		fields.ElementMap{
			"git_path":     paasArgoGitPath,
			"git_revision": updatedRevision,
			"git_url":      paasArgoGitURL,
			"paas":         paasWithArgo,
			"requestor":    paasRequestor,
			"service":      "paas",
			"subservice":   "capability",
		},
		"ApplicationSet List generator contains the correct parameters",
	)

	assert.NotNil(
		t,
		getOrFail(ctx, paasArgoNs, corev1.NamespaceAll, &corev1.Namespace{}, t, cfg),
		"ArgoCD namespace created",
	)

	return ctx
}

func assertArgoCRB(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	defaultRoleBinding := getOrFail(ctx, "paas-monitoring-edit", "", &rbac.ClusterRoleBinding{}, t, cfg)

	subjects := defaultRoleBinding.Subjects
	assert.Len(t, subjects, 2, "ClusterRoleBinding contains two subjects")
	var subjectNames []string
	for _, subject := range subjects {
		assert.Equal(t, "ServiceAccount", subject.Kind, "Subject is of type ServiceAccount")
		assert.Equal(t, paasArgoNs, subject.Namespace, "Subject is from correct namespace")
		subjectNames = append(subjectNames, subject.Name)
	}
	assert.Contains(t, subjectNames, "argo-service-applicationset-controller", "ClusterRoleBinding contains")
	assert.Contains(t, subjectNames, "argo-service-argocd-application-controller", "ClusterRoleBinding contains")

	extraRoleBinding := getOrFail(ctx, "paas-admin", "", &rbac.ClusterRoleBinding{}, t, cfg)
	assert.Len(t, extraRoleBinding.Subjects, 1, "ClusterRoleBinding contains one subject")
	assert.Equal(t, paasArgoNs, extraRoleBinding.Subjects[0].Namespace, "Subject is from correct namespace")
	assert.Equal(
		t,
		"argo-service-argocd-application-controller",
		extraRoleBinding.Subjects[0].Name,
		"Subject is as defined in capability extra_permissions",
	)

	return ctx
}

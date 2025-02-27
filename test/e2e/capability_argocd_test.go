package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"
	"github.com/belastingdienst/opr-paas/internal/stubs/argoproj-labs/v1beta1"
	argo "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"

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
	paasWithArgo        = "paas-capability-argocd"
	paasArgoNs          = paasWithArgo + "-argocd"
	paasArgoGitURL      = "ssh://git@scm/some-repo.git"
	paasArgoGitPath     = "foo/"
	paasArgoGitRevision = "main"
	paasRequestor       = "paas-requestor"
	// String `dummysecret` encrypted with fixtures/crypt/pub/publicKey0
	// revive:disable-next-line
	paasArgoSecret = "mPNADW4KlAYmiBSXfgyoP6G0h/8prFQNH7VBFXB3xiZ8wij2sRIgKekVUC3N9cHk73wkuewoH2fyM0BH2P1xKvSP4v4wwzq+fJC6qxx+d/lucrfnBHWCpsAr646OVYyoH8Er6PpBrPxM+OXCjVsXhd/8CGA32VzcUKSrAWBVWTgXpJ4/X/9gez865AmZkfFf2WBImYgs5Q/rH/mPP1jxl3WP10g51FLi4XG1qn2XdLRzBKXRKluh+PvMRYgqZ8QKl2Yd2HWj1SkzXrtayB7197r0fQ6t4cwpn8mqy30GQhsw6NEPSkcYakukOX2PYeRIVCwmMl3uEe9X1y7fesQVBMnq1loQJRpd7kBUj6EErnKNZ9Qa8tOXYLMME2tzsaYWz+rxhczCaMv9r55EGBENRB0K6VMY4jfC4NKkcVwgZm182/Z1wzOnPbhSKAoaSYUXVrsNfjuzlvQGJmaNF4onDgJdVpqJxkEH98E3q+NMlSYhIzZDph1RDjHmUm2aoAhx2W9zle+LsOWHLgogPHRwY+N7NRII5SBEnw99miCAQVqHnpEk0uITzny0G5AuoS9aKmVhbUNNR1TgZ6u2dFjrkbnZB0GKilJhVENM+oE8Fbq7Q4Qa9wtk/GK1myPNvY7ARbw1tfvbcpJT/NtKnEKsho/OVzfHn15W3niNVpXrZgs=" //nolint:gosec
)

func TestCapabilityArgoCD(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: paasRequestor,
		Quota:     quota.Quota{},
		Capabilities: api.PaasCapabilities{
			"argocd": api.PaasCapability{
				CustomFields: map[string]string{
					"git_revision": paasArgoGitRevision,
				},
				Enabled:          true,
				SSHSecrets:       map[string]string{paasArgoGitURL: paasArgoSecret},
				GitURL:           paasArgoGitURL,
				GitPath:          paasArgoGitPath,
				ExtraPermissions: true,
			},
		},
	}

	testenv.Test(
		t,
		features.New("ArgoCD Capability").
			Setup(createPaasFn(paasWithArgo, paasSpec)).
			Assess("ArgoCD application is created", assertArgoCDCreated).
			Assess("ArgoCD application is updated", assertArgoCDUpdated).
			Assess("ArgoCD application has ClusterRoleBindings", assertArgoCRB).
			Teardown(teardownPaasFn(paasWithArgo)).
			Feature(),
	)
}

func assertArgoCDCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithArgo, t, cfg)
	argopaasns := &api.PaasNS{ObjectMeta: metav1.ObjectMeta{
		Name:      "argocd",
		Namespace: paasWithArgo,
	}}
	require.NoError(
		t,
		waitForCondition(ctx, cfg, argopaasns, 0, api.TypeReadyPaasNs),
		"ArgoCD PaasNS reconciliation succeeds",
	)

	argoAppSet := getOrFail(ctx, "argoas", "asns", &argo.ApplicationSet{}, t, cfg)
	entries, _ := getApplicationSetListEntries(argoAppSet)

	assert.Len(t, entries, 1, "ApplicationSet contains one List generator")
	assert.Equal(t, map[string]string{
		"git_path":     paasArgoGitPath,
		"git_revision": paasArgoGitRevision,
		"git_url":      paasArgoGitURL,
		"paas":         paasWithArgo,
		"requestor":    paasRequestor,
		"service":      "paas",
		"subservice":   "capability",
	}, entries[0], "ApplicationSet List generator contains the correct parameters")

	assert.NotNil(
		t,
		getOrFail(ctx, paasArgoNs, corev1.NamespaceAll, &corev1.Namespace{}, t, cfg),
		"ArgoCD namespace created",
	)

	// Assert ArgoCD creation
	argocd := getOrFail(ctx, "argocd", paasArgoNs, &v1beta1.ArgoCD{}, t, cfg)
	assert.Equal(t, paas.UID, argocd.OwnerReferences[0].UID)
	assert.Equal(t, "role:tester", *argocd.Spec.RBAC.DefaultPolicy)
	assert.Equal(t, "g, system:cluster-admins, role:admin", *argocd.Spec.RBAC.Policy)
	assert.Equal(t, "[groups]", *argocd.Spec.RBAC.Scopes)

	applications := listOrFail(ctx, paasArgoNs, &argo.ApplicationList{}, t, cfg).Items
	assert.Len(t, applications, 1, "An application is present in the ArgoCD namespace")
	assert.Equal(t, "paas-bootstrap", applications[0].Name)
	assert.Equal(t, argo.ApplicationSource{
		RepoURL:        paasArgoGitURL,
		Path:           paasArgoGitPath,
		TargetRevision: paasArgoGitRevision,
	}, *applications[0].Spec.Source, "Application source matches Git properties from Paas")
	assert.Equal(
		t,
		"whatever",
		applications[0].Spec.IgnoreDifferences[0].Name,
		"`exclude_appset_name` configuration is included in IgnoreDifferences",
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
		corev1.ResourceLimitsCPU:                                  resource.MustParse("5"),
		corev1.ResourceRequestsCPU:                                resource.MustParse("1"),
		corev1.ResourceLimitsMemory:                               resource.MustParse("4Gi"),
		corev1.ResourceRequestsMemory:                             resource.MustParse("1Gi"),
		corev1.ResourceRequestsStorage:                            resource.MustParse("0"),
		"thin.storageclass.storage.k8s.io/persistentvolumeclaims": resource.MustParse("0"),
	}, crq.Spec.Quota.Hard, "Quota conforms to defaults from Paas config")

	return ctx
}

func assertArgoCDUpdated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	updatedRevision := "updatedRevision"
	paas := getPaas(ctx, paasWithArgo, t, cfg)
	paas.Spec.Capabilities = api.PaasCapabilities{
		"argocd": api.PaasCapability{
			Enabled:          true,
			SSHSecrets:       map[string]string{paasArgoGitURL: paasArgoSecret},
			GitURL:           paasArgoGitURL,
			GitPath:          paasArgoGitPath,
			GitRevision:      updatedRevision,
			ExtraPermissions: true,
		},
	}

	// As only the Paas spec is updated via the above change, we wait for that and
	// know that no reconciliation of PaasNs takes place so no need to wait for that.
	// check #185 for more details
	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}
	argoAppSet := getOrFail(ctx, "argoas", "asns", &argo.ApplicationSet{}, t, cfg)
	entries, _ := getApplicationSetListEntries(argoAppSet)

	// For now this still applies, later we move the git_.. properties to the appSet as well
	// Assert AppSet entry updated accordingly
	assert.Len(t, entries, 1, "ApplicationSet contains one List generator")
	assert.Equal(t, map[string]string{
		"git_path":     paasArgoGitPath,
		"git_revision": updatedRevision,
		"git_url":      paasArgoGitURL,
		"paas":         paasWithArgo,
		"requestor":    paasRequestor,
		"service":      "paas",
		"subservice":   "capability",
	}, entries[0], "ApplicationSet List generator contains the correct parameters")

	assert.NotNil(
		t,
		getOrFail(ctx, paasArgoNs, corev1.NamespaceAll, &corev1.Namespace{}, t, cfg),
		"ArgoCD namespace created",
	)

	// Assert ArgoCD unchanged
	argocd := getOrFail(ctx, "argocd", paasArgoNs, &v1beta1.ArgoCD{}, t, cfg)
	assert.Equal(t, paas.UID, argocd.OwnerReferences[0].UID)
	assert.Equal(t, "role:tester", *argocd.Spec.RBAC.DefaultPolicy)
	assert.Equal(t, "g, system:cluster-admins, role:admin", *argocd.Spec.RBAC.Policy)
	assert.Equal(t, "[groups]", *argocd.Spec.RBAC.Scopes)

	// Assert Bootstrap is now updated as described in issue #185
	applications := listOrFail(ctx, paasArgoNs, &argo.ApplicationList{}, t, cfg).Items
	assert.Len(t, applications, 1, "An application is present in the ArgoCD namespace")
	assert.Equal(t, "paas-bootstrap", applications[0].Name)
	assert.Equal(t, argo.ApplicationSource{
		RepoURL:        paasArgoGitURL,
		Path:           paasArgoGitPath,
		TargetRevision: updatedRevision,
	}, *applications[0].Spec.Source, "Application source matches Git properties from Paas")
	assert.Equal(
		t,
		"whatever",
		applications[0].Spec.IgnoreDifferences[0].Name,
		"`exclude_appset_name` configuration is included in IgnoreDifferences",
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

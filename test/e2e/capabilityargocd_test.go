package e2e

import (
	"context"
	"testing"

	"github.com/belastingdienst/opr-paas/internal/stubs/argoproj-labs/v1beta1"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"
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
	paasArgoGitUrl      = "ssh://git@scm/some-repo.git"
	paasArgoGitPath     = "foo/"
	paasArgoGitRevision = "main"
	// String `dummysecret` encrypted with fixtures/crypt/pub/publicKey0
	paasArgoSecret = "mPNADW4KlAYmiBSXfgyoP6G0h/8prFQNH7VBFXB3xiZ8wij2sRIgKekVUC3N9cHk73wkuewoH2fyM0BH2P1xKvSP4v4wwzq+fJC6qxx+d/lucrfnBHWCpsAr646OVYyoH8Er6PpBrPxM+OXCjVsXhd/8CGA32VzcUKSrAWBVWTgXpJ4/X/9gez865AmZkfFf2WBImYgs5Q/rH/mPP1jxl3WP10g51FLi4XG1qn2XdLRzBKXRKluh+PvMRYgqZ8QKl2Yd2HWj1SkzXrtayB7197r0fQ6t4cwpn8mqy30GQhsw6NEPSkcYakukOX2PYeRIVCwmMl3uEe9X1y7fesQVBMnq1loQJRpd7kBUj6EErnKNZ9Qa8tOXYLMME2tzsaYWz+rxhczCaMv9r55EGBENRB0K6VMY4jfC4NKkcVwgZm182/Z1wzOnPbhSKAoaSYUXVrsNfjuzlvQGJmaNF4onDgJdVpqJxkEH98E3q+NMlSYhIzZDph1RDjHmUm2aoAhx2W9zle+LsOWHLgogPHRwY+N7NRII5SBEnw99miCAQVqHnpEk0uITzny0G5AuoS9aKmVhbUNNR1TgZ6u2dFjrkbnZB0GKilJhVENM+oE8Fbq7Q4Qa9wtk/GK1myPNvY7ARbw1tfvbcpJT/NtKnEKsho/OVzfHn15W3niNVpXrZgs=" //nolint:gosec
)

func TestCapabilityArgoCD(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: "paas-requestor",
		Quota:     quota.Quotas{},
		Capabilities: api.PaasCapabilities{
			"argocd": api.PaasCapability{
				Enabled:          true,
				SshSecrets:       map[string]string{paasArgoGitUrl: paasArgoSecret},
				GitUrl:           paasArgoGitUrl,
				GitPath:          paasArgoGitPath,
				GitRevision:      paasArgoGitRevision,
				ExtraPermissions: true,
			},
		},
	}

	testenv.Test(
		t,
		features.New("ArgoCD Capability").
			Setup(createPaasFn(paasWithArgo, paasSpec)).
			Assess("ArgoCD application is created", assertArgoCDCreated).
			Assess("ArgoCD application has ClusterRoleBindings", assertArgoCRB).
			Teardown(teardownPaasFn(paasWithArgo)).
			Feature(),
	)
}

func assertArgoCDCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithArgo, t, cfg)
	argopaasns := getOrFail(ctx, "argocd", paasWithArgo, &api.PaasNS{}, t, cfg)
	require.NoError(t, waitForPaasNSReconciliation(ctx, cfg, argopaasns), "ArgoCD PaasNS reconciliation succeeds")

	argoAppSet := getOrFail(ctx, "argoas", "asns", &argo.ApplicationSet{}, t, cfg)
	entries, _ := getApplicationSetListEntries(argoAppSet)

	assert.Len(t, entries, 1, "ApplicationSet contains one List generator")
	assert.Equal(t, map[string]string{
		"paas":       paasWithArgo,
		"requestor":  "paas-requestor",
		"service":    "paas",
		"subservice": "capability",
	}, entries[0], "ApplicationSet List generator contains the correct parameters")

	assert.NotNil(t, getOrFail(ctx, paasArgoNs, corev1.NamespaceAll, &corev1.Namespace{}, t, cfg), "ArgoCD namespace created")

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
		RepoURL:        paasArgoGitUrl,
		Path:           paasArgoGitPath,
		TargetRevision: paasArgoGitRevision,
	}, *applications[0].Spec.Source, "Application source matches Git properties from Paas")
	assert.Equal(t, "whatever", applications[0].Spec.IgnoreDifferences[0].Name, "`exclude_appset_name` configuration is included in IgnoreDifferences")

	secrets := listOrFail(ctx, paasArgoNs, &corev1.SecretList{}, t, cfg).Items
	assert.Len(t, secrets, 1)
	assert.Equal(t, "dummysecret", string(secrets[0].Data["sshPrivateKey"]), "SSH secret is created in ArgoCD namespace")

	crq := getOrFail(ctx, paasArgoNs, corev1.NamespaceAll, &quotav1.ClusterResourceQuota{}, t, cfg)
	assert.Equal(t, "q.lbl="+paasArgoNs, metav1.FormatLabelSelector(crq.Spec.Selector.LabelSelector), "Quota selects ArgoCD namespace via selector set to `quota_label` configuration")
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

/*
Default_permissions points:

1. Assess that a clusterrolebinding for `paas-monitoring-edit` is created
2. Assess that the `paas-monitoring-edit` clusterrolebinding contains the `argo-service-applicationset-controller` service account
3. Assess that the `paas-monitoring-edit` clusterrolebinding contains the `argo-service-argocd-application-controller` service account
*/
func assertArgoCRB(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	argo_role_binding := getOrFail(ctx, "paas-monitoring-edit", "", &rbac.ClusterRoleBinding{}, t, cfg)

	subjects := argo_role_binding.Subjects
	assert.Len(t, subjects, 2, "ClusterRoleBinding contains one subject")
	var subjectNames []string
	for _, subject := range subjects {
		assert.Equal(t, "ServiceAccount", subject.Kind, "Subject is of type ServiceAccount")
		assert.Equal(t, paasArgoNs, subject.Namespace, "Subject is from correct namespace")
		subjectNames = append(subjectNames, subject.Name)
	}
	assert.Contains(t, subjectNames, "argo-service-applicationset-controller", "ClusterRoleBinding contains")
	assert.Contains(t, subjectNames, "argo-service-argocd-applicationset-controller", "ClusterRoleBinding contains")
	return ctx
}

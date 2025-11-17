package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	userv1 "github.com/openshift/api/user/v1"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	GroupNameFormat          = "%s-%s"
	paasWithGroupQuery       = "paas-group-query"
	paasGroupQueryNamespace  = "group-query-ns"
	paasGroupQueryAbsoluteNs = paasWithGroupQuery + "-" + paasGroupQueryNamespace
	groupWithQueryName       = "aug-cet-groupquery" //nolint:gosec
	groupQuery               = "CN=aug-cet-groupquery,OU=paas,DC=test,DC=acme,DC=org"
	group2Query              = "CN=aug-cet-queryviewrole,OU=paas,DC=test,DC=acme,DC=org"
)

func TestGroupQuery(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor:  "paas-user",
		Namespaces: api.PaasNamespaces{paasGroupQueryNamespace: api.PaasNamespace{}},
		Quota: map[corev1.ResourceName]resource.Quantity{
			"cpu":    resource.MustParse("200m"),
			"memory": resource.MustParse("256Mi"),
		},
		Groups:     api.PaasGroups{groupWithQueryName: api.PaasGroup{Query: groupQuery}},
	}

	testenv.Test(
		t,
		features.New("Group with query").
			Setup(createPaasFn(paasWithGroupQuery, paasSpec)).
			Assess("query group is reconciled correctly", assertQueryGroupReconciled).
			Assess("second query group is reconciled after Paas update", assertQueryGroupReconciledAfterUpdate).
			Assess("works when key and CN are different", assertGroupKeyAndNameDifferenceIsOk).
			Assess("first group remains unchanged after Paas update", assertQueryGroupReconciled).
			Teardown(teardownPaasFn(paasWithGroupQuery)).
			Feature(),
	)
}

func assertQueryGroupReconciled(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroupQuery, t, cfg)
	rolebinding := getOrFail(ctx, "paas-admin", paasGroupQueryAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroupQuery, &rbacv1.RoleBindingList{}, t, cfg)

	// No LDAP group should be created
	failWhenExists(ctx, paas.GroupKey2GroupName(groupWithQueryName), cfg.Namespace(), &userv1.Group{}, t, cfg)

	assert.Len(t, rolebinding.Subjects, 1)
	assert.Equal(t, paas.GroupKey2GroupName(groupWithQueryName), rolebinding.Subjects[0].Name,
		"The configured default RoleBinding should be set for the group")
	assert.Equal(t, "admin", rolebinding.RoleRef.Name,
		"The role in the Paas `rolemappings` configuration for the default role should be applied in the RoleBinding")
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(t, paas.GroupKey2GroupName(groupWithQueryName), sub.Name,
				"No RoleBindings should be set on the parent Paas")
		}
	}

	return ctx
}

func assertQueryGroupReconciledAfterUpdate(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroupQuery, t, cfg)
	group2Key := "aug-cet-queryviewrole"
	paas.Spec.Groups[group2Key] = api.PaasGroup{
		Query: group2Query,
		Users: []string{"foo", "bar"},
		Roles: []string{"viewer"},
	}

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	// No LDAP group should be created
	failWhenExists(ctx, paas.GroupKey2GroupName(group2Key), cfg.Namespace(), &userv1.Group{}, t, cfg)

	rolebinding := getOrFail(ctx, "paas-view", paasGroupQueryAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroupQuery, &rbacv1.RoleBindingList{}, t, cfg)

	assert.Len(t, rolebinding.Subjects, 1)
	assert.Equal(t, paas.GroupKey2GroupName(group2Key), rolebinding.Subjects[0].Name,
		"A RoleBinding for the passed 'viewer' role should be set for the group")
	assert.Equal(t, "view", rolebinding.RoleRef.Name,
		//revive:disable-next-line
		"The role in the Paas `rolemappings` configuration for the passed 'viewer' role should be applied in the RoleBinding")
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(t, paas.GroupKey2GroupName(group2Key), sub.Name,
				"No RoleBindings should be set on the parent Paas")
		}
	}

	return ctx
}

func assertGroupKeyAndNameDifferenceIsOk(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroupQuery, t, cfg)
	group3Key := "aug-cet-group3key"
	paas.Spec.Groups[group3Key] = api.PaasGroup{
		Query: "CN=different",
		Roles: []string{"viewer"},
	}

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	// No LDAP group should be created
	failWhenExists(ctx, group3Key, cfg.Namespace(), &userv1.Group{}, t, cfg)
	failWhenExists(ctx, paas.GroupKey2GroupName(group3Key), cfg.Namespace(), &userv1.Group{}, t, cfg)
	rolebinding := getOrFail(ctx, "paas-view", paasGroupQueryAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroupQuery, &rbacv1.RoleBindingList{}, t, cfg)

	expectedSubject := rbacv1.Subject{
		Kind:     "Group",
		APIGroup: "rbac.authorization.k8s.io", Name: paas.GroupKey2GroupName(group3Key),
	}
	assert.Contains(t, rolebinding.Subjects, expectedSubject,
		"A RoleBinding for the passed 'viewer' role should be set for the group")
	assert.Equal(t, "view", rolebinding.RoleRef.Name,
		//revive:disable-next-line
		"The role in the Paas `rolemappings` configuration for the passed 'viewer' role should be applied in the RoleBinding")
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(
				t,
				paas.GroupKey2GroupName(group3Key),
				sub.Name,
				"No RoleBindings should be set on the parent Paas",
			)
		}
	}

	return ctx
}

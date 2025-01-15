package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"

	userv1 "github.com/openshift/api/user/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasWithGroupQuery       = "paas-group-query"
	paasGroupQueryNamespace  = "group-query-ns"
	paasGroupQueryAbsoluteNs = paasWithGroupQuery + "-" + paasGroupQueryNamespace
	groupWithQueryName       = "aug-cet-groupquery" //nolint:gosec
	groupQuery               = "CN=aug-cet-groupquery,OU=paas,DC=test,DC=acme,DC=org"
	group2Query              = "CN=aug-cet-queryviewrole,OU=paas,DC=test,DC=acme,DC=org"
	updatedGroup2Query       = "CN=aug-cet-second-queryviewrole,OU=paas,DC=test,DC=acme,DC=org"
)

func TestGroupQuery(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor:  "paas-user",
		Namespaces: []string{paasGroupQueryNamespace},
		Quota:      make(quota.Quota),
		Groups:     api.PaasGroups{groupWithQueryName: api.PaasGroup{Query: groupQuery}},
	}

	testenv.Test(
		t,
		features.New("Group with query").
			Setup(createPaasFn(paasWithGroupQuery, paasSpec)).
			Assess("group is created with query", assertGroupQueryCreated).
			Assess("groupsynclist contains the group query", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				groupsynclist := getOrFail(ctx, "wlname", "gsns", &corev1.ConfigMap{}, t, cfg)
				assert.Equal(t, groupQuery, groupsynclist.Data["groupsynclist.txt"], "The groupsynclist includes the group query")

				return ctx
			}).
			Assess("second group with role and query is created after Paas update", assertGroupQueryCreatedAfterUpdate).
			Assess("groupsynclist contains both group's queries", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				groupsynclist := getOrFail(ctx, "wlname", "gsns", &corev1.ConfigMap{}, t, cfg)
				assert.Equal(t, groupQuery+"\n"+group2Query, groupsynclist.Data["groupsynclist.txt"], "The groupsynclist should include both group's queries")

				return ctx
			}).
			Assess("old group is removed from groupsynclist when groupkey is renamed", assertLdapGroupRemovedAfterUpdatingKey).
			Assess("first group remains unchanged after Paas update", assertGroupQueryCreated).
			Assess("groups are deleted when Paas is deleted", assertGroupQueryDeleted).
			Teardown(teardownPaasFn(paasWithGroupQuery)).
			Feature(),
	)
}

func assertGroupQueryCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroupQuery, t, cfg)
	group := getOrFail(ctx, groupWithQueryName, cfg.Namespace(), &userv1.Group{}, t, cfg)
	rolebinding := getOrFail(ctx, "paas-admin", paasGroupQueryAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroupQuery, &rbacv1.RoleBindingList{}, t, cfg)

	assert.Equal(t, groupWithQueryName, group.Name, "The group name should match the one defined in the Paas")
	assert.Empty(t, group.Users, "No users should be defined in the group")
	assert.Len(t, group.Labels, 1)
	assert.Equal(t, "my-ldap-host", group.Labels["openshift.io/ldap.host"], "The correct label should be defined")
	assert.Equal(t, paas.UID, group.OwnerReferences[0].UID, "The owner of the group should be the Paas defining it")
	assert.Len(t, rolebinding.Subjects, 1)
	assert.Equal(t, groupWithQueryName, rolebinding.Subjects[0].Name, "The configured default RoleBinding should be set for the group")
	assert.Equal(t, "admin", rolebinding.RoleRef.Name, "The role in the Paas `rolemappings` configuration for the default role should be applied in the RoleBinding")
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(t, groupWithQueryName, sub.Name, "No RoleBindings should be set on the parent Paas")
		}
	}

	return ctx
}

func assertGroupQueryCreatedAfterUpdate(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroupQuery, t, cfg)
	group2Name := "aug-cet-queryviewrole"
	paas.Spec.Groups[group2Name] = api.PaasGroup{
		Query: group2Query,
		Roles: []string{"viewer"},
	}

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	group2 := getOrFail(ctx, group2Name, cfg.Namespace(), &userv1.Group{}, t, cfg)
	rolebinding := getOrFail(ctx, "paas-view", paasGroupQueryAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroupQuery, &rbacv1.RoleBindingList{}, t, cfg)

	assert.Equal(t, group2Name, group2.Name, "The group name should match the one defined in the Paas")
	assert.Empty(t, group2.Users, "No users should be defined in the group")
	assert.Len(t, group2.Labels, 1, "Group should contain one label")
	assert.Equal(t, "my-ldap-host", group2.Labels["openshift.io/ldap.host"], "The ldap.host label should contain PaasConfig value")
	assert.Len(t, group2.Annotations, 2, "Group should have 2 annotations")
	assert.Equal(t, group2Query, group2.Annotations["openshift.io/ldap.uid"], "The ldap.uid annotation should contain group.query value")
	assert.Equal(t, "my-ldap-host:13", group2.Annotations["openshift.io/ldap.url"], "The ldap.url annotation should contain PaasConfig value")
	assert.Equal(t, paas.UID, group2.OwnerReferences[0].UID, "The owner of the group should be the Paas defining it")
	assert.Len(t, rolebinding.Subjects, 1)
	assert.Equal(t, group2Name, rolebinding.Subjects[0].Name, "A RoleBinding for the passed 'viewer' role should be set for the group")
	assert.Equal(t, "view", rolebinding.RoleRef.Name, "The role in the Paas `rolemappings` configuration for the passed 'viewer' role should be applied in the RoleBinding")
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(t, group2Name, sub.Name, "No RoleBindings should be set on the parent Paas")
		}
	}

	return ctx
}

func assertLdapGroupRemovedAfterUpdatingKey(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroupQuery, t, cfg)
	paas.Spec.Groups = api.PaasGroups{groupWithQueryName: api.PaasGroup{Query: groupQuery}, "updatedSecondLdapGroup": api.PaasGroup{
		Query: updatedGroup2Query,
		Roles: []string{"viewer"},
	}}

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	// Regression for #269 old group should be removed from groupsynclist
	groupsynclist := getOrFail(ctx, "wlname", "gsns", &corev1.ConfigMap{}, t, cfg)
	t.Log("Groupsynclist", groupsynclist)
	assert.NotContains(t, groupsynclist.Data["groupsynclist.txt"], group2Query,
		"The groupsynclist does not include obsolete group query")

	return ctx
}

func assertGroupQueryDeleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaasSync(ctx, paasWithGroupQuery, t, cfg)
	groups := listOrFail(ctx, "", &userv1.GroupList{}, t, cfg)

	assert.Empty(t, groups.Items, "k8s should not return any groups")

	return ctx
}

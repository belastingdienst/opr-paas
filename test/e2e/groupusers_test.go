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
	paasWithGroups         = "paas-with-groups"
	paasNamespace          = "my-ns"
	paasAbsoluteNs         = paasWithGroups + "-" + paasNamespace
	groupName              = "aug-cet-test"
	secondGroupName        = "aug-cet-viewrole"
	updatedSecondGroupName = "aug-cet-viewrole-updated"
)

func TestGroupUsers(t *testing.T) {
	groups := api.PaasGroups{groupName: api.PaasGroup{Users: []string{"foo"}}}
	paasSpec := api.PaasSpec{
		Requestor:  "paas-user",
		Namespaces: []string{"my-ns"},
		Quota:      make(quota.Quota),
		Groups:     groups,
	}

	testenv.Test(
		t,
		features.New("Group with users").
			Setup(createPaasFn(paasWithGroups, paasSpec)).
			Assess("group is created with user", assertGroupCreated).
			Assess("second group with role is created after Paas update", assertGroupCreatedAfterUpdate).
			Assess("old group is not removed when group in Paas is renamed", assertGroupNotRemovedAfterUpdatingKey).
			Assess("first group remains unchanged after Paas update", assertGroupCreated).
			Assess("is deleted when Paas is deleted", assertGroupsDeleted).
			Teardown(teardownPaasFn(paasWithGroups)).
			Feature(),
	)
}

func assertGroupCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroups, t, cfg)
	group := getOrFail(ctx, groupName, cfg.Namespace(), &userv1.Group{}, t, cfg)
	groupsynclist := getOrFail(ctx, "wlname", "gsns", &corev1.ConfigMap{}, t, cfg)
	rolebinding := getOrFail(ctx, "paas-admin", paasAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroups, &rbacv1.RoleBindingList{}, t, cfg)

	// Group name matches the one defined in the Paas
	assert.Equal(t, groupName, group.Name)
	// Defined user is in group
	assert.Equal(t, "[foo]", group.Users.String())
	// Correct labels are defined
	assert.Len(t, group.Labels, 1)
	assert.Equal(t, "my-ldap-host", group.Labels["openshift.io/ldap.host"])
	assert.Len(t, group.Annotations, 2, "Group should have 2 annotations")
	assert.Equal(t, "", group.Annotations["openshift.io/ldap.uid"], "The ldap.uid annotation should contain group.query value")
	assert.Equal(t, "my-ldap-host:13", group.Annotations["openshift.io/ldap.url"], "The ldap.url annotation should contain PaasConfig value")
	// The owner of the group is the Paas that created it
	assert.Equal(t, paas.UID, group.OwnerReferences[0].UID)
	// The groupsynclist is unchanged (empty)
	assert.Empty(t, groupsynclist.Data["groupsynclist.txt"])
	// Default RoleBinding (as per Paas config) is set for group
	assert.Len(t, rolebinding.Subjects, 1)
	assert.Equal(t, groupName, rolebinding.Subjects[0].Name)
	assert.Equal(t, "admin", rolebinding.RoleRef.Name)
	// No RoleBindings set on parent Paas
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(t, groupName, sub.Name)
		}
	}

	return ctx
}

func assertGroupCreatedAfterUpdate(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroups, t, cfg)
	paas.Spec.Groups[secondGroupName] = api.PaasGroup{
		Roles: []string{"viewer"},
		Users: []string{"bar"},
	}

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	group2 := getOrFail(ctx, secondGroupName, cfg.Namespace(), &userv1.Group{}, t, cfg)
	groupsynclist := getOrFail(ctx, "wlname", "gsns", &corev1.ConfigMap{}, t, cfg)
	rolebinding := getOrFail(ctx, "paas-view", paasAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroups, &rbacv1.RoleBindingList{}, t, cfg)

	// Group name matches the one defined in the Paas
	assert.Equal(t, secondGroupName, group2.Name)
	// Defined user is in group
	assert.Equal(t, "[bar]", group2.Users.String())
	// Correct labels are defined
	assert.Len(t, group2.Labels, 1)
	assert.Equal(t, "my-ldap-host", group2.Labels["openshift.io/ldap.host"])
	// The owner of the group is the Paas that created it
	assert.Equal(t, paas.UID, group2.OwnerReferences[0].UID)
	// The groupsynclist is unchanged (empty)
	assert.Empty(t, groupsynclist.Data["groupsynclist.txt"])
	// Default RoleBinding (as per Paas config) is set for group
	assert.Len(t, rolebinding.Subjects, 1)
	assert.Equal(t, secondGroupName, rolebinding.Subjects[0].Name)
	assert.Equal(t, "view", rolebinding.RoleRef.Name)
	// No RoleBindings set on parent Paas
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(t, secondGroupName, sub.Name)
		}
	}

	return ctx
}

func assertGroupNotRemovedAfterUpdatingKey(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroups, t, cfg)
	paas.Spec.Groups = api.PaasGroups{groupName: api.PaasGroup{Users: []string{"foo"}}, updatedSecondGroupName: api.PaasGroup{
		Roles: []string{"viewer"},
		Users: []string{"bar"},
	}}

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	// Regression for #269 "old" group still in k8s
	secondGroup := getOrFail(ctx, secondGroupName, cfg.Namespace(), &userv1.Group{}, t, cfg)
	// Updated groupname created
	updatedGroup2 := getOrFail(ctx, updatedSecondGroupName, cfg.Namespace(), &userv1.Group{}, t, cfg)
	groupsynclist := getOrFail(ctx, "wlname", "gsns", &corev1.ConfigMap{}, t, cfg)
	rolebinding := getOrFail(ctx, "paas-view", paasAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroups, &rbacv1.RoleBindingList{}, t, cfg)

	// Group name matches the one defined in the Paas before updating it.
	assert.Equal(t, secondGroupName, secondGroup.Name)
	// Group name matches the one defined in the Paas
	assert.Equal(t, updatedSecondGroupName, updatedGroup2.Name)
	// Defined user is in group
	assert.Equal(t, "[bar]", updatedGroup2.Users.String())
	// Correct labels are defined
	assert.Len(t, updatedGroup2.Labels, 1)
	assert.Equal(t, "my-ldap-host", updatedGroup2.Labels["openshift.io/ldap.host"])
	// The owner of the group is the Paas that created it
	assert.Equal(t, paas.UID, updatedGroup2.OwnerReferences[0].UID)
	// The groupsynclist is unchanged (empty)
	assert.Empty(t, groupsynclist.Data["groupsynclist.txt"])
	// Regression for #269 "old" rolebinding is removed
	// Default RoleBinding (as per Paas config) is set for group
	assert.Len(t, rolebinding.Subjects, 1)
	assert.Equal(t, updatedSecondGroupName, rolebinding.Subjects[0].Name)
	assert.Equal(t, "view", rolebinding.RoleRef.Name)
	// No RoleBindings set on parent Paas
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(t, updatedSecondGroupName, sub.Name)
		}
	}

	return ctx
}

func assertGroupsDeleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaasSync(ctx, paasWithGroups, t, cfg)
	groups := listOrFail(ctx, "", &userv1.GroupList{}, t, cfg)

	assert.Empty(t, groups.Items)

	return ctx
}

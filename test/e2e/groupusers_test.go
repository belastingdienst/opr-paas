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
	paasWithGroups = "paas-with-groups"
	paasNamespace  = "my-ns"
	paasAbsoluteNs = paasWithGroups + "-" + paasNamespace
	groupName      = "aug-cet-test"
)

func TestGroupUsers(t *testing.T) {
	groups := api.PaasGroups{groupName: api.PaasGroup{Users: []string{"foo"}}}
	paasSpec := api.PaasSpec{
		Requestor:  "paas-user",
		Namespaces: []string{"my-ns"},
		Quota:      make(quota.Quotas),
		Groups:     groups,
	}

	testenv.Test(
		t,
		features.New("Group with users").
			Setup(createPaasFn(paasWithGroups, paasSpec)).
			Assess("group is created with user", assertGroupCreated).
			Assess("second group with role is created after Paas update", assertGroupCreatedAfterUpdate).
			Assess("first group remains unchanged after Paas update", assertGroupCreated).
			Assess("is deleted when Paas is deleted", assertGroupsDeleted).
			Teardown(teardownPaasFn(paasWithGroups)).
			Feature(),
	)
}

func assertGroupCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroups, t, cfg)
	group := getOrFail(ctx, groupName, cfg.Namespace(), &userv1.Group{}, t, cfg)
	whitelist := getOrFail(ctx, "wlname", "wlns", &corev1.ConfigMap{}, t, cfg)
	rolebinding := getOrFail(ctx, "paas-admin", paasAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroups, &rbacv1.RoleBindingList{}, t, cfg)

	// Group name matches the one defined in the Paas
	assert.Equal(t, groupName, group.Name)
	// Defined user is in group
	assert.Equal(t, "[foo]", group.Users.String())
	// Correct labels are defined
	assert.Len(t, group.Labels, 1)
	assert.Equal(t, "my-ldap-host", group.Labels["openshift.io/ldap.host"])
	// The owner of the group is the Paas that created it
	assert.Equal(t, paas.UID, group.OwnerReferences[0].UID)
	// The whitelist is unchanged (empty)
	assert.Empty(t, whitelist.Data["whitelist.txt"])
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
	group2Name := "aug-cet-viewrole"
	paas.Spec.Groups[group2Name] = api.PaasGroup{
		Roles: []string{"viewer"},
		Users: []string{"bar"},
	}

	if err := cfg.Client().Resources().Update(ctx, paas); err != nil {
		t.Fatalf("Failed to update Paas resource: %v", err)
	}

	waitForOperator()

	group2 := getOrFail(ctx, group2Name, cfg.Namespace(), &userv1.Group{}, t, cfg)
	whitelist := getOrFail(ctx, "wlname", "wlns", &corev1.ConfigMap{}, t, cfg)
	rolebinding := getOrFail(ctx, "paas-view", paasAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroups, &rbacv1.RoleBindingList{}, t, cfg)

	// Group name matches the one defined in the Paas
	assert.Equal(t, group2Name, group2.Name)
	// Defined user is in group
	assert.Equal(t, "[bar]", group2.Users.String())
	// Correct labels are defined
	assert.Len(t, group2.Labels, 1)
	assert.Equal(t, "my-ldap-host", group2.Labels["openshift.io/ldap.host"])
	// The owner of the group is the Paas that created it
	assert.Equal(t, paas.UID, group2.OwnerReferences[0].UID)
	// The whitelist is unchanged (empty)
	assert.Empty(t, whitelist.Data["whitelist.txt"])
	// Default RoleBinding (as per Paas config) is set for group
	assert.Len(t, rolebinding.Subjects, 1)
	assert.Equal(t, group2Name, rolebinding.Subjects[0].Name)
	assert.Equal(t, "view", rolebinding.RoleRef.Name)
	// No RoleBindings set on parent Paas
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(t, group2Name, sub.Name)
		}
	}

	return ctx
}

func assertGroupsDeleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaas(ctx, paasWithGroups, t, cfg)
	groups := listOrFail(ctx, "", &userv1.GroupList{}, t, cfg)

	assert.Empty(t, groups.Items)

	return ctx
}

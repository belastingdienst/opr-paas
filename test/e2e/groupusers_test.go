package e2e

import (
	"context"
	"fmt"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/controller"
	"github.com/belastingdienst/opr-paas/internal/quota"

	userv1 "github.com/openshift/api/user/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasWithGroups        = "paas-with-groups"
	paasNamespace         = "my-ns"
	paasAbsoluteNs        = paasWithGroups + "-" + paasNamespace
	groupKey              = "aug-cet-test"
	secondGroupKey        = "aug-cet-viewrole"
	updatedSecondGroupKey = "aug-cet-viewrole-updated"
	groupSyncListName     = "groupsynclist.txt"
)

func TestGroupUsers(t *testing.T) {
	groups := api.PaasGroups{groupKey: api.PaasGroup{Users: []string{"foo"}}}
	paasSpec := api.PaasSpec{
		Requestor:  paasRequestor,
		Namespaces: []string{paasNamespace},
		Quota:      make(quota.Quota),
		Groups:     groups,
	}

	testenv.Test(
		t,
		features.New("Group with users").
			Setup(createPaasFn(paasWithGroups, paasSpec)).
			Assess("group is created with user", assertGroupCreated).
			Assess("second group with role is created after Paas update", assertGroupCreatedAfterUpdate).
			Assess("old group is removed when group in Paas is renamed", assertOldGroupRemovedAfterUpdatingKey).
			Assess("first group remains unchanged after Paas update", assertGroupCreated).
			Assess("is deleted when Paas is deleted", assertGroupsDeleted).
			Teardown(teardownPaasFn(paasWithGroups)).
			Feature(),
	)
}

func assertGroupCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroups, t, cfg)
	group := getOrFail(ctx, paas.GroupKey2GroupName(groupKey), cfg.Namespace(), &userv1.Group{}, t, cfg)
	groupsynclist := getOrFail(ctx, "wlname", "gsns", &corev1.ConfigMap{}, t, cfg)
	rolebinding := getOrFail(ctx, "paas-admin", paasAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroups, &rbacv1.RoleBindingList{}, t, cfg)

	assert.Equal(t, paas.GroupKey2GroupName(groupKey), group.Name)
	// Defined user is in group
	assert.Equal(t, "[foo]", group.Users.String())
	// Correct labels are defined
	assert.Len(t, group.Labels, 1)
	assert.Equal(t, paas.Name, group.Labels["app.kubernetes.io/managed-by"], "Labeled as managed by Paas")
	assert.Empty(t, group.Annotations, "Group should have no annotations")
	// The owner of the group is the Paas that created it
	assert.Equal(t, paas.UID, group.OwnerReferences[0].UID)
	// The groupsynclist is unchanged (empty)
	assert.Empty(t, groupsynclist.Data[groupSyncListName])
	// Default RoleBinding (as per Paas config) is set for group
	assert.Len(t, rolebinding.Subjects, 1)
	assert.Equal(t, paas.GroupKey2GroupName(groupKey), rolebinding.Subjects[0].Name)
	assert.Equal(t, "admin", rolebinding.RoleRef.Name)
	// No RoleBindings set on parent Paas
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(t, paas.GroupKey2GroupName(groupKey), sub.Name)
		}
	}

	return ctx
}

func assertGroupCreatedAfterUpdate(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroups, t, cfg)
	paas.Spec.Groups[secondGroupKey] = api.PaasGroup{
		Roles: []string{"viewer"},
		Users: []string{"bar"},
	}

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	group2 := getOrFail(ctx, paas.GroupKey2GroupName(secondGroupKey), cfg.Namespace(), &userv1.Group{}, t, cfg)
	groupsynclist := getOrFail(ctx, "wlname", "gsns", &corev1.ConfigMap{}, t, cfg)
	rolebinding := getOrFail(ctx, "paas-view", paasAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroups, &rbacv1.RoleBindingList{}, t, cfg)

	assert.Equal(t, paas.GroupKey2GroupName(secondGroupKey), group2.Name)
	// Defined user is in group
	assert.Equal(t, "[bar]", group2.Users.String())
	// Correct labels are defined
	assert.Len(t, group2.Labels, 1)
	assert.Equal(
		t,
		paas.Name,
		group2.Labels[controller.ManagedByLabelKey],
		"Labeled as managed by Paas.name",
	)
	// The owner of the group is the Paas that created it
	assert.Equal(t, paas.UID, group2.OwnerReferences[0].UID)
	// The groupsynclist is unchanged (empty)
	assert.Empty(t, groupsynclist.Data[groupSyncListName])
	// Default RoleBinding (as per Paas config) is set for group
	assert.Len(t, rolebinding.Subjects, 1)
	assert.Equal(t, fmt.Sprintf(GroupNameFormat, paasWithGroups, secondGroupKey), rolebinding.Subjects[0].Name)
	assert.Equal(t, "view", rolebinding.RoleRef.Name)
	// No RoleBindings set on parent Paas
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(t, fmt.Sprintf(GroupNameFormat, paasWithGroups, secondGroupKey), sub.Name)
		}
	}

	return ctx
}

func assertOldGroupRemovedAfterUpdatingKey(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroups, t, cfg)
	paas.Spec.Groups = api.PaasGroups{
		groupKey: api.PaasGroup{Users: []string{"foo"}},
		updatedSecondGroupKey: api.PaasGroup{
			Roles: []string{"viewer"},
			Users: []string{"bar"},
		},
	}

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	failWhenExists(ctx, paas.GroupKey2GroupName(secondGroupKey), cfg.Namespace(), &userv1.Group{}, t, cfg)
	// Updated groupname created
	updatedGroup2 := getOrFail(
		ctx,
		paas.GroupKey2GroupName(updatedSecondGroupKey),
		cfg.Namespace(),
		&userv1.Group{},
		t,
		cfg,
	)
	groupsynclist := getOrFail(ctx, "wlname", "gsns", &corev1.ConfigMap{}, t, cfg)
	rolebinding := getOrFail(ctx, "paas-view", paasAbsoluteNs, &rbacv1.RoleBinding{}, t, cfg)
	rolebindingsPaas := listOrFail(ctx, paasWithGroups, &rbacv1.RoleBindingList{}, t, cfg)

	// Group name matches the one defined in the Paas
	assert.Equal(t, paas.GroupKey2GroupName(updatedSecondGroupKey), updatedGroup2.Name)
	// Defined user is in group
	assert.Equal(t, "[bar]", updatedGroup2.Users.String())
	// Correct labels are defined
	assert.Len(t, updatedGroup2.Labels, 1)
	assert.NotContains(t, controller.LdapHostLabelKey, updatedGroup2.Labels)
	assert.Equal(
		t,
		paasWithGroups,
		updatedGroup2.Labels[controller.ManagedByLabelKey],
		"Labeled as managed by Paas.name",
	)
	// The owner of the group is the Paas that created it
	assert.Equal(t, paas.UID, updatedGroup2.OwnerReferences[0].UID)
	// The groupsynclist is unchanged (empty)
	assert.Empty(t, groupsynclist.Data[groupSyncListName])
	// Default RoleBinding (as per Paas config) is set for group
	assert.Len(t, rolebinding.Subjects, 1)
	assert.Equal(t, paas.GroupKey2GroupName(updatedSecondGroupKey), rolebinding.Subjects[0].Name)
	assert.Equal(t, "view", rolebinding.RoleRef.Name)
	// No RoleBindings set on parent Paas
	for _, rb := range rolebindingsPaas.Items {
		for _, sub := range rb.Subjects {
			assert.NotEqual(t, paas.GroupKey2GroupName(updatedSecondGroupKey), sub.Name)
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

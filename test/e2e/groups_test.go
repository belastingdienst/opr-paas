package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"

	userv1 "github.com/openshift/api/user/v1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasWithGroups = "paas-with-groups"
	groupName      = "aug-cet-test"
)

func TestGroups(t *testing.T) {
	groups := api.PaasGroups{groupName: api.PaasGroup{Users: []string{"foo"}}}
	paasSpec := api.PaasSpec{
		Requestor: "paas-user",
		Quota:     make(quota.Quotas),
		Groups:    groups,
	}

	testenv.Test(
		t,
		features.New("Groups").
			Setup(createPaasFn(paasWithGroups, paasSpec)).
			Assess("group is created", assertGroupCreated).
			Assess("second group with role is created after PaaS update", assertGroupCreatedAfterUpdate).
			Assess("first group remains unchanged after PaaS update", assertGroupCreated).
			Assess("is deleted when PaaS is deleted", assertGroupsDeleted).
			Teardown(teardownPaasFn(paasWithGroups)).
			Feature(),
	)
}

func assertGroupCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroups, t, cfg)
	group := getGroup(ctx, groupName, t, cfg)

	// Group name matches the one defined in the PaaS
	assert.Equal(t, groupName, group.Name)
	// Defined user is in group
	assert.Equal(t, "[foo]", group.Users.String())
	// Correct labels are defined
	assert.Len(t, group.Labels, 1)
	assert.Equal(t, "my-ldap-host", group.Labels["openshift.io/ldap.host"])
	// The owner of the group is the PaaS that created it
	assert.Equal(t, paas.UID, group.OwnerReferences[0].UID)

	return ctx
}

func assertGroupCreatedAfterUpdate(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithGroups, t, cfg)
	paas.Spec.Groups["aug-cet-viewrole"] = api.PaasGroup{
		Roles: []string{"view"},
	}

	if err := cfg.Client().Resources().Update(ctx, &paas); err != nil {
		t.Fatalf("Failed to update PaaS resource: %v", err)
	}

	waitForOperator()

	group2 := getGroup(ctx, "aug-cet-viewrole", t, cfg)

	// Group name matches the one defined in the PaaS
	assert.Equal(t, "aug-cet-viewrole", group2.Name)

	return ctx
}

func assertGroupsDeleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	deletePaas(ctx, paasWithGroups, t, cfg)

	var groups userv1.GroupList

	if err := cfg.Client().Resources().List(ctx, &groups); err != nil {
		t.Fatalf("Failed to retrieve Group: %v", err)
	}

	assert.Empty(t, groups.Items)

	return ctx
}

func getGroup(ctx context.Context, name string, t *testing.T, cfg *envconf.Config) userv1.Group {
	var group userv1.Group

	if err := cfg.Client().Resources().Get(ctx, name, cfg.Namespace(), &group); err != nil {
		t.Fatalf("Failed to retrieve Group: %v", err)
	}

	return group
}

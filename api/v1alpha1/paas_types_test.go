/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testGroups = PaasGroups{
	"cn=test1": PaasGroup{
		Query: "CN=test2,OU=org_unit,DC=example,DC=org",
		Users: []string{"usr1", "usr2"},
	},
	"cn=test3": PaasGroup{
		Query: "CN=test4",
		Users: []string{"usr3", "usr2"},
	},
}

// Paas

func TestPaas_PrefixedBoolMap(t *testing.T) {
	paas := Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: "paas",
		},
	}

	input := map[string]bool{
		"test": true,
		"smt":  false,
	}

	output := paas.PrefixedBoolMap(input)

	assert.NotNil(t, output)
	assert.IsType(t, map[string]bool{}, output)
	assert.Contains(t, output, "paas-test")
	assert.Contains(t, output, "paas-smt")
	assert.NotContains(t, output, "paas-doesntexist")
	assert.True(t, output["paas-test"])
	assert.False(t, output["paas-smt"])
}

func TestPaas_GetNsSshSecrets(t *testing.T) {
	paas := Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: "paas",
		},
		Spec: PaasSpec{
			Namespaces: []string{"argocd"},
			Capabilities: PaasCapabilities{
				"argocd": PaasCapability{
					Enabled:    true,
					SshSecrets: map[string]string{"capsecret1": "capsecretvalue1"},
				},
			},
			SshSecrets: map[string]string{"key1": "value1", "key2": "value2"},
		},
	}

	output := paas.GetNsSshSecrets("nonexistingNS")
	assert.NotNil(t, output)
	assert.IsType(t, map[string]string{}, output)
	assert.Contains(t, output, "key1")
	assert.NotContains(t, output, "capsecret1")
	assert.Equal(t, "value1", output["key1"])

	output = paas.GetNsSshSecrets("argocd")
	assert.NotNil(t, output)
	assert.IsType(t, map[string]string{}, output)
	assert.Contains(t, output, "key1")
	assert.Contains(t, output, "capsecret1")
	assert.Equal(t, "value1", output["key1"])
	assert.Equal(t, "capsecretvalue1", output["capsecret1"])
}

// PaasGroups

func TestPaasGroups_Filtered(t *testing.T) {
	pgs := PaasGroups{
		"grp1": {},
		"grp2": {},
		"grp3": {},
	}

	// Nothing to filter
	output := pgs.Filtered([]string{})
	assert.IsType(t, PaasGroups{}, output)
	assert.Equal(t, pgs, output)

	// Filtered one group
	output = pgs.Filtered([]string{"grp2"})
	assert.IsType(t, PaasGroups{}, output)
	assert.NotEqual(t, pgs, output)
	assert.NotContains(t, output, "grp1")
	assert.Contains(t, output, "grp2")
	assert.NotContains(t, output, "grp3")

	// Filtered two groups
	output = pgs.Filtered([]string{"grp1", "grp3"})
	assert.IsType(t, PaasGroups{}, output)
	assert.NotEqual(t, pgs, output)
	assert.Contains(t, output, "grp1")
	assert.NotContains(t, output, "grp2")
	assert.Contains(t, output, "grp3")
}

func TestPaasGroups_Roles(t *testing.T) {
	pgs := PaasGroups{
		"grp1": {},
		"grp2": {
			Roles: []string{},
		},
		"grp3": {
			Roles: []string{
				"grp3-role1",
			},
		},
		"grp4": {
			Roles: []string{
				"grp4-role1",
				"grp4-role2",
				"grp4-role3",
			},
		},
	}

	// Nothing to filter
	output := pgs.Roles()
	assert.NotNil(t, output)
	assert.IsType(t, map[string][]string{}, output)
	assert.Contains(t, output, "grp1")
	assert.Contains(t, output, "grp2")
	assert.Contains(t, output, "grp3")
	assert.Contains(t, output, "grp4")
	assert.Empty(t, output["grp1"])
	assert.NotEmpty(t, output["grp3"])
	assert.Len(t, output["grp3"], 1)
	assert.Contains(t, output["grp3"], "grp3-role1")
	assert.Len(t, output["grp4"], 3)
	assert.Contains(t, output["grp4"], "grp4-role1")
	assert.Contains(t, output["grp4"], "grp4-role2")
	assert.Contains(t, output["grp4"], "grp4-role3")
}

func TestPaasGroups_Key2Name(t *testing.T) {
	assert.NotNil(t, "", testGroups.Key2Name("cn=test123"))
	assert.Equal(t, "test2", testGroups.Key2Name("cn=test1"))
	assert.Equal(t, "", testGroups.Key2Name("cn=test123"))
	assert.Equal(t, "test4", testGroups.Key2Name("cn=test3"))
}

func TestPaasGroups_Names(t *testing.T) {
	output := testGroups.Names()
	assert.NotNil(t, output)
	assert.Len(t, output, 2)
	assert.Contains(t, output, "test2")
	assert.Contains(t, output, "test4")
}

func TestPaasGroups_LdapQueries(t *testing.T) {
	ldapGroups := testGroups.LdapQueries()
	sort.Strings(ldapGroups)
	assert.Len(t, ldapGroups, 2)
	assert.Equal(t, "CN=test2,OU=org_unit,DC=example,DC=org", ldapGroups[0])
	assert.Equal(t, "CN=test4", ldapGroups[1])
}

func TestPaasGroups_AsGroups(t *testing.T) {
	assert.NotNil(t, testGroups.AsGroups())
}

// PaasCapabilities

func TestPaasCapabilities_AsPrefixedMap(t *testing.T) {
	pc := PaasCapabilities{
		"argocd":  PaasCapability{},
		"grafana": PaasCapability{},
	}

	// Empty prefix
	output := pc.AsPrefixedMap("")

	assert.NotNil(t, output)
	assert.IsType(t, PaasCapabilities{}, output)
	assert.Contains(t, output, "argocd")
	assert.Contains(t, output, "grafana")

	// "test" prefix
	output = pc.AsPrefixedMap("test")

	assert.NotNil(t, output)
	assert.IsType(t, PaasCapabilities{}, output)
	assert.Contains(t, output, "test-argocd")
	assert.Contains(t, output, "test-grafana")
}

func TestPaasCapabilities_CapExtraFields(t *testing.T) {
	var pc PaasCapability
	var fields map[string]string
	var errs []error

	// argocd specific fields can come from old and new options
	// new options over old options
	// validation success works as expected
	// key not being set which is not required defaults to config.Default
	pc = PaasCapability{
		GitUrl:  "https://github.com/org/repo",
		GitPath: "argocd/myconfig",
		CustomFields: map[string]string{
			"git_url":      "https://github.com/org/other-repo",
			"git_revision": "develop",
		},
	}
	fields, errs = pc.CapExtraFields(map[string]ConfigCustomField{
		"git_url":      {Validation: "^https://.*$"},
		"git_revision": {},
		"git_path":     {},
		"default_key":  {Default: "default_value"},
	})
	assert.Empty(t, errs, "we should have no errors returned")
	assert.Equal(t, map[string]string{
		"git_url":      "https://github.com/org/other-repo",
		"git_revision": "develop",
		"git_path":     "argocd/myconfig",
		"default_key":  "default_value",
	}, fields)

	// key not in config throws error
	pc = PaasCapability{
		CustomFields: map[string]string{
			"not_in_config": "breaks",
		},
	}
	fields, errs = pc.CapExtraFields(map[string]ConfigCustomField{})
	assert.Len(t, errs, 1, "not_in_config should return 1 error")
	assert.Nil(t, fields, "not_in_config should return nilmap for fields")

	// required_field key not being set throws error
	pc = PaasCapability{
		CustomFields: map[string]string{},
	}
	fields, errs = pc.CapExtraFields(map[string]ConfigCustomField{
		"required_key": {Required: true},
	})
	assert.Len(t, errs, 1, "required_field should return 1 error")
	assert.Nil(t, fields, "required_field should return nilmap for fields")

	// validation errors throw issues
	pc = PaasCapability{
		CustomFields: map[string]string{
			"invalid_key": "invalid_value",
		},
	}
	fields, errs = pc.CapExtraFields(map[string]ConfigCustomField{
		"invalid_key": {
			Validation: "^valid_value$",
		},
	})
	assert.Len(t, errs, 1, "invalid_value should return 1 error")
	assert.Nil(t, fields, "invalid_value should return nilmap for fields")
}

func TestPaasCapabilities_IsCap(t *testing.T) {
	pc := PaasCapabilities{
		"argocd": PaasCapability{
			Enabled: true,
		},
		"grafana": PaasCapability{
			Enabled: false,
		},
		"ci": PaasCapability{},
	}

	// Empty prefix
	// assert.Fail(t, "TEST", fmt.Sprintf("%v", pc.AsMap()))
	assert.True(t, pc.IsCap("argocd"))
	assert.False(t, pc.IsCap("grafana"))
	assert.False(t, pc.IsCap("ci"))
	assert.False(t, pc.IsCap("sso"))
}

// PaasCapability

func TestPaasCapability_SetDefaults(t *testing.T) {
	pa := PaasCapability{
		GitRevision: "",
		GitPath:     "",
	}

	pa.SetDefaults()
	assert.Equal(t, ".", pa.GitPath)
	assert.Equal(t, "master", pa.GitRevision)

	pa.GitPath = "/test"
	pa.GitRevision = "main"

	pa.SetDefaults()
	assert.Equal(t, "/test", pa.GitPath)
	assert.Equal(t, "main", pa.GitRevision)
}

// PaasStatus

func TestPaasStatus_Truncate(t *testing.T) {
	ps := PaasStatus{
		Messages: []string{
			"test msg 1",
			"test msg 2",
		},
	}

	assert.IsType(t, []string{}, ps.Messages)
	assert.Len(t, ps.Messages, 2)
	assert.Contains(t, ps.Messages, "test msg 1")

	ps.Truncate()
	assert.IsType(t, []string{}, ps.Messages)
	assert.Empty(t, ps.Messages)
	assert.NotContains(t, ps.Messages, "test msg 1")
}

// Paas

func Test_Paas_ClonedAnnotations(t *testing.T) {
	paas := Paas{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"key 1": "value 1",
				"key 2": "value 2",
				"key 3": "value 3",
			},
		},
	}

	output := paas.ClonedAnnotations()

	assert.NotNil(t, output)
	assert.IsType(t, map[string]string{}, output)
	assert.Len(t, output, 3)
	assert.Contains(t, output, "key 1")
	assert.Contains(t, output, "key 2")
	assert.Contains(t, output, "key 3")
	assert.Equal(t, "value 1", output["key 1"])
}

func Test_Paas_ClonedLabels(t *testing.T) {
	paas := Paas{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"key 1":                      "value 1",
				"app.kubernetes.io/instance": "value 2",
				"key 3":                      "value 3",
			},
		},
	}

	output := paas.ClonedLabels()

	assert.NotNil(t, output)
	assert.IsType(t, map[string]string{}, output)
	assert.Len(t, output, 2)
	assert.Contains(t, output, "key 1")
	assert.NotContains(t, output, "app.kubernetes.io/instance")
	assert.Contains(t, output, "key 3")
	assert.Equal(t, "value 1", output["key 1"])
}

func Test_Paas_IsItMe(t *testing.T) {
	paas := Paas{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MyKind",
			APIVersion: "1.1.1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "Some Name",
		},
	}

	test1 := metav1.OwnerReference{
		Kind:       "MyKind",
		APIVersion: "1.1.1",
		Name:       "Some Name",
	}

	test2 := metav1.OwnerReference{
		Kind:       "MyOtherKind",
		APIVersion: "1.1.1",
		Name:       "Some Name",
	}

	test3 := metav1.OwnerReference{
		Kind:       "MyKind",
		APIVersion: "1.1.0",
		Name:       "Some Name",
	}

	test4 := metav1.OwnerReference{
		Kind:       "MyKind",
		APIVersion: "1.1.1",
		Name:       "Some Other Name",
	}

	test5 := metav1.OwnerReference{
		Kind:       "MyOtherKind",
		APIVersion: "1.1.0",
		Name:       "Some Name",
	}

	test6 := metav1.OwnerReference{}

	assert.True(t, paas.IsItMe(test1))
	assert.False(t, paas.IsItMe(test2))
	assert.False(t, paas.IsItMe(test3))
	assert.False(t, paas.IsItMe(test4))
	assert.False(t, paas.IsItMe(test5))
	assert.False(t, paas.IsItMe(test6))
}

func Test_Paas_AmIOwner(t *testing.T) {
	paas := Paas{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MyKind",
			APIVersion: "1.1.1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "Some Name",
		},
	}

	ref1 := metav1.OwnerReference{
		Kind:       "MyKind",
		APIVersion: "1.1.1",
		Name:       "Some Name",
	}

	ref2 := metav1.OwnerReference{
		Kind:       "MyOtherKind",
		APIVersion: "1.1.1",
		Name:       "Some Name",
	}

	owner := []metav1.OwnerReference{
		ref1,
		ref2,
	}
	notOwner := []metav1.OwnerReference{
		ref2,
		ref2,
	}

	empty := []metav1.OwnerReference{}

	assert.True(t, paas.AmIOwner(owner))
	assert.False(t, paas.AmIOwner(notOwner))
	assert.False(t, paas.AmIOwner(empty))
}

func Test_Paas_WithoutMe(t *testing.T) {
	paas := Paas{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MyKind",
			APIVersion: "1.1.1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "Some Name",
		},
	}

	ref1 := metav1.OwnerReference{
		Kind:       "MyKind",
		APIVersion: "1.1.1",
		Name:       "Some Name",
	}

	ref2 := metav1.OwnerReference{
		Kind:       "MyOtherKind",
		APIVersion: "1.1.1",
		Name:       "Some Name",
	}

	owner := []metav1.OwnerReference{
		ref1,
		ref2,
	}
	notOwner := []metav1.OwnerReference{
		ref2,
		ref2,
	}

	empty := []metav1.OwnerReference{}

	assert.NotContains(t, paas.WithoutMe(owner), ref1)
	assert.Contains(t, paas.WithoutMe(owner), ref2)
	assert.NotContains(t, paas.WithoutMe(notOwner), ref1)
	assert.Contains(t, paas.WithoutMe(notOwner), ref2)
	assert.Empty(t, paas.WithoutMe(empty))
}

// compound tests

func TestPaas_Namespaces(t *testing.T) {
	paas := Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: "paas-test",
		},
		Spec: PaasSpec{
			Namespaces: []string{
				"argocd",
				"sso",
				"extra",
			},
			Capabilities: PaasCapabilities{
				"argocd": PaasCapability{
					Enabled: true,
				},
				"grafana": PaasCapability{
					Enabled: true,
				},
				"sso": PaasCapability{
					Enabled: false,
				},
			},
		},
	}
	enCapNs := paas.enabledCapNamespaces()
	assert.NotNil(t, enCapNs)
	assert.IsType(t, map[string]bool{}, enCapNs)
	assert.Contains(t, enCapNs, "argocd")
	assert.Contains(t, enCapNs, "grafana")
	assert.NotContains(t, enCapNs, "sso")
	assert.NotContains(t, enCapNs, "extra")

	enExNs := paas.extraNamespaces()
	assert.NotNil(t, enExNs)
	assert.IsType(t, map[string]bool{}, enExNs)
	assert.NotContains(t, enExNs, "argocd")
	assert.NotContains(t, enExNs, "grafana")
	assert.NotContains(t, enExNs, "sso")
	assert.Contains(t, enExNs, "extra")

	enEnNs := paas.AllEnabledNamespaces()
	assert.NotNil(t, enEnNs)
	assert.IsType(t, map[string]bool{}, enEnNs)
	assert.Contains(t, enEnNs, "argocd")
	assert.Contains(t, enEnNs, "grafana")
	assert.NotContains(t, enEnNs, "sso")
	assert.Contains(t, enEnNs, "extra")
}

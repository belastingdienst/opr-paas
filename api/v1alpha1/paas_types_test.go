/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"
	"sort"
	"testing"

	"github.com/belastingdienst/opr-paas/internal/fields"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	grp1         = "grp1"
	grp2         = "grp2"
	grp3         = "grp3"
	grp4         = "grp4"
	tstGroup     = "testGroup"
	paasName     = "paasName"
	test1        = "test1"
	test2        = "test2"
	test3        = "test3"
	test4        = "test4"
	cntest1      = "cn=" + test1
	cntest2      = "cn=" + test2
	cntest3      = "cn=" + test3
	cntest4      = "cn=" + test4
	argoCap      = "argocd"
	grafanaCap   = "grafana"
	ciCap        = "tekton"
	ssoCap       = "keycloak"
	extraCap     = "extra"
	myKind       = "MyKind"
	otherKind    = "MyOtherKind"
	myVersion    = "1.1.1"
	otherVersion = "1.1.0"
	myName       = "Some Name"
	otherName    = "Some Other Name"
)

var testGroups = PaasGroups{
	cntest1: PaasGroup{
		Query: cntest2 + ",OU=org_unit,DC=example,DC=org",
		Users: []string{"usr1", "usr2"},
	},
	cntest3: PaasGroup{
		Query: cntest4,
		Users: []string{"usr3", "usr2"},
	},
	tstGroup: PaasGroup{
		Users: []string{"usr3", "usr2"},
	},
}

// Paas

func TestPaas_PrefixedBoolMap(t *testing.T) {
	paas := Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
	}

	input := map[string]bool{
		"test": true,
		"smt":  false,
	}

	output := paas.PrefixedBoolMap(input)

	assert.NotNil(t, output)
	assert.IsType(t, map[string]bool{}, output)
	for k, v := range input {
		assert.Contains(t, output, join(paasName, k))
		assert.Equal(t, v, output[join(paasName, k)])
	}
	assert.NotContains(t, output, join(paasName, "doesntexist"))
}

func TestPaas_GetNsSSHSecrets(t *testing.T) {
	const (
		capSecretName    = "capsecret1"
		capSecretValue   = "capSecretValue"
		paasSecretName1  = "paasSecret1"
		paasSecretValue1 = "paasSecretValue1"
		paasSecretName2  = "paasSecret2"
		paasSecretValue2 = "paasSecretValue1"
	)
	paas := Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
		Spec: PaasSpec{
			Namespaces: []string{argoCap},
			Capabilities: PaasCapabilities{
				argoCap: PaasCapability{
					Enabled:    true,
					SSHSecrets: map[string]string{capSecretName: capSecretValue},
				},
			},
			SSHSecrets: map[string]string{paasSecretName1: paasSecretValue1, paasSecretName2: paasSecretValue2},
		},
	}

	output := paas.GetNsSSHSecrets("nonexistingNS")
	assert.NotNil(t, output)
	assert.IsType(t, map[string]string{}, output)
	assert.Contains(t, output, paasSecretName1)
	assert.NotContains(t, output, capSecretName)
	assert.Equal(t, paasSecretValue1, output[paasSecretName1])

	output = paas.GetNsSSHSecrets(argoCap)
	assert.NotNil(t, output)
	assert.IsType(t, map[string]string{}, output)
	assert.Contains(t, output, paasSecretName1)
	assert.Contains(t, output, capSecretName)
	assert.Equal(t, paasSecretValue1, output[paasSecretName1])
	assert.Equal(t, capSecretValue, output[capSecretName])
}

// PaasGroups

func TestPaasGroups_Filtered(t *testing.T) {
	pgs := PaasGroups{
		grp1: {
			Query: "cn=test1,ou=org_unit,dc=example,dc=org",
		},
		grp2: {},
		grp3: {},
	}

	// Nothing to filter
	output := pgs.Filtered([]string{})
	assert.IsType(t, PaasGroups{}, output)
	assert.Equal(t, pgs, output)

	// Filtered one group
	output = pgs.Filtered([]string{grp2})
	assert.IsType(t, PaasGroups{}, output)
	assert.NotEqual(t, pgs, output)
	assert.NotContains(t, output, grp1)
	assert.Contains(t, output, grp2)
	assert.NotContains(t, output, grp3)

	// Filtered two groups
	output = pgs.Filtered([]string{grp1, grp3})
	assert.IsType(t, PaasGroups{}, output)
	assert.NotEqual(t, pgs, output)
	assert.Contains(t, output, grp1)
	assert.NotContains(t, output, grp2)
	assert.Contains(t, output, grp3)
}

func TestPaasGroups_Roles(t *testing.T) {
	pgs := PaasGroups{
		grp1: {},
		grp2: {
			Query: "CN=test1,OU=org_unit,DC=example,DC=org",
			Roles: []string{},
		},
		grp3: {
			Roles: []string{
				"grp3-role1",
			},
		},
		grp4: {
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
	assert.Contains(t, output, grp1)
	assert.Contains(t, output, grp2)
	assert.Contains(t, output, grp3)
	assert.Contains(t, output, grp4)
	assert.Empty(t, output[grp1])
	assert.NotEmpty(t, output[grp3])
	assert.Len(t, output[grp3], 1)
	assert.Contains(t, output[grp3], "grp3-role1")
	assert.Len(t, output[grp4], 3)
	assert.Contains(t, output[grp4], "grp4-role1")
	assert.Contains(t, output[grp4], "grp4-role2")
	assert.Contains(t, output[grp4], "grp4-role3")
}

func TestPaasGroups_Key2Name(t *testing.T) {
	const cntest123 = "cn=test123"
	assert.NotNil(t, testGroups.Key2Name(cntest123))
	assert.Equal(t, test2, testGroups.Key2Name(cntest1))
	assert.Equal(t, "", testGroups.Key2Name(cntest123))
	assert.Equal(t, test4, testGroups.Key2Name(cntest3))
}

func TestPaasGroups_Keys(t *testing.T) {
	assert.NotNil(t, testGroups.Keys(), "Keys not nill")
	assert.Contains(t, testGroups.Keys(), cntest1)
	assert.Contains(t, testGroups.Keys(), cntest3)
	assert.NotContains(t, testGroups.Keys(), cntest4)
}

func TestPaas_GroupKey2GroupName(t *testing.T) {
	paas := Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
		Spec: PaasSpec{
			Groups: testGroups,
		},
	}
	assert.Equal(t, "", paas.GroupKey2GroupName("testers"), "Key not present in Paas")
	assert.NotNil(t, paas.GroupKey2GroupName(cntest1), "")
	assert.Equal(
		t,
		test2,
		paas.GroupKey2GroupName(cntest1),
		cntest1+" is a query group thus returning Key2Name value.",
	)
	assert.NotNil(t, paas.GroupKey2GroupName(tstGroup), "Test is a present")
	assert.Equal(t, join(paasName, tstGroup), paas.GroupKey2GroupName(tstGroup),
		"Test is a group of users thus prefixed by Paas name")
}

func TestPaas_GroupNames(t *testing.T) {
	paas := Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
		Spec: PaasSpec{
			Groups: testGroups,
		},
	}
	groupNames := paas.GroupNames()
	assert.Len(t, groupNames, 3, "Three group names found")
	assert.Contains(t, groupNames, test2, cntest1+" is a query group thus returning Key2Name value.")
	assert.Contains(t, groupNames, join(paasName, tstGroup), "Test is a group of users thus prefixed by Paas name")
	assert.Contains(t, groupNames, test4, cntest3+" is a query group thus returning Key2Name value.")
}

func TestPaasGroups_Names(t *testing.T) {
	output := testGroups.Names()
	assert.NotNil(t, output)
	assert.Len(t, output, 3)
	assert.Contains(t, output, tstGroup)
	assert.Contains(t, output, test2)
	assert.Contains(t, output, test4)
}

func TestPaasGroups_LdapQueries(t *testing.T) {
	ldapGroups := testGroups.LdapQueries()
	sort.Strings(ldapGroups)
	assert.Len(t, ldapGroups, 2)
	assert.Equal(t, cntest2+",OU=org_unit,DC=example,DC=org", ldapGroups[0])
	assert.Equal(t, cntest4, ldapGroups[1])
}

func TestPaasGroups_AsGroups(t *testing.T) {
	assert.NotNil(t, testGroups.AsGroups())
}

// PaasCapabilities

func TestPaasCapabilities_AsPrefixedMap(t *testing.T) {
	pc := PaasCapabilities{
		argoCap:    PaasCapability{},
		grafanaCap: PaasCapability{},
	}

	// Empty prefix
	output := pc.AsPrefixedMap("")

	assert.NotNil(t, output)
	assert.IsType(t, PaasCapabilities{}, output)
	assert.Contains(t, output, argoCap)
	assert.Contains(t, output, grafanaCap)

	prefix := "test"
	output = pc.AsPrefixedMap(prefix)

	assert.NotNil(t, output)
	assert.IsType(t, PaasCapabilities{}, output)
	assert.Contains(t, output, join(prefix, argoCap))
	assert.Contains(t, output, join(prefix, grafanaCap))
}

func TestPaasCapabilities_CapExtraFields(t *testing.T) {
	var pc PaasCapability
	var elements fields.Element
	var err error

	// argocd specific fields can come from old and new options
	// new options over old options
	// validation success works as expected
	// key not being set which is not required defaults to config.Default
	pc = PaasCapability{
		GitURL:  "https://github.com/org/repo",
		GitPath: "argocd/myconfig",
		CustomFields: map[string]string{
			"git_url":      "https://github.com/org/other-repo",
			"git_revision": "develop",
		},
	}
	elements, err = pc.CapExtraFields(map[string]ConfigCustomField{
		"git_url":      {Validation: "^https://.*$"},
		"git_revision": {},
		"git_path":     {},
		"default_key":  {Default: "default_value"},
	})
	require.NoError(t, err, "we should have no errors returned")
	assert.Equal(t, fields.Elements{
		"git_url":      "https://github.com/org/other-repo",
		"git_revision": "develop",
		"git_path":     "argocd/myconfig",
		"default_key":  "default_value",
	}, elements)

	// key not in config throws error
	pc = PaasCapability{
		CustomFields: map[string]string{
			"not_in_config": "breaks",
		},
	}
	elements, err = pc.CapExtraFields(map[string]ConfigCustomField{})
	assert.Equal(t, "custom field not_in_config is not configured in capability config", err.Error())
	assert.Nil(t, elements, "not_in_config should return nilmap for fields")

	// required_field key not being set throws error
	pc = PaasCapability{
		CustomFields: map[string]string{},
	}
	elements, err = pc.CapExtraFields(map[string]ConfigCustomField{
		"required_key": {Required: true},
	})
	assert.Equal(t, "value required_key is required", err.Error())
	assert.Nil(t, elements, "required_field should return nilmap for fields")

	// validation errors throw issues
	pc = PaasCapability{
		CustomFields: map[string]string{
			"invalid_key": "invalid_value",
		},
	}
	elements, err = pc.CapExtraFields(map[string]ConfigCustomField{
		"invalid_key": {
			Validation: "^valid_value$",
		},
	})
	assert.Equal(t, "invalid value invalid_value (does not match ^valid_value$)", err.Error())
	assert.Nil(t, elements, "invalid_value should return nilmap for fields")
}

func TestPaasCapabilities_IsCap(t *testing.T) {
	pc := PaasCapabilities{
		argoCap: PaasCapability{
			Enabled: true,
		},
		grafanaCap: PaasCapability{
			Enabled: false,
		},
		ciCap: PaasCapability{},
	}

	// Empty prefix
	// assert.Fail(t, "TEST", fmt.Sprintf("%v", pc.AsMap()))
	assert.True(t, pc.IsCap(argoCap))
	assert.False(t, pc.IsCap(grafanaCap))
	assert.False(t, pc.IsCap(ciCap))
	assert.False(t, pc.IsCap(ssoCap))
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
	msg1 := "test msg 1"
	msg2 := "test msg 2"
	ps := PaasStatus{
		Messages: []string{
			msg1,
			msg2,
		},
	}

	assert.IsType(t, []string{}, ps.Messages)
	assert.Len(t, ps.Messages, 2)
	assert.Contains(t, ps.Messages, msg1)

	ps.Truncate()
	assert.IsType(t, []string{}, ps.Messages)
	assert.Empty(t, ps.Messages)
	assert.NotContains(t, ps.Messages, msg1)
}

// Paas

func Test_Paas_ClonedAnnotations(t *testing.T) {
	paas := Paas{}
	paas.Annotations = make(map[string]string)
	for i := 0; i < 3; i++ {
		paas.Annotations[fmt.Sprintf("key %d", i)] = fmt.Sprintf("value %d", i)
	}

	output := paas.ClonedAnnotations()

	assert.NotNil(t, output)
	assert.IsType(t, map[string]string{}, output)
	assert.Len(t, output, 3)
	for i := 0; i < 3; i++ {
		assert.Contains(t, output, fmt.Sprintf("key %d", i))
		assert.Equal(t, fmt.Sprintf("value %d", i), output[fmt.Sprintf("key %d", i)])
	}
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

func generateReferences() (refs []metav1.OwnerReference) {
	for _, kind := range []string{myKind, otherKind} {
		for _, version := range []string{myVersion, otherVersion} {
			for _, name := range []string{myName, otherName} {
				refs = append(refs,
					metav1.OwnerReference{
						Kind:       kind,
						APIVersion: version,
						Name:       name,
					})
			}
		}
	}
	return refs
}

func Test_Paas_IsItMe(t *testing.T) {
	allOwners := generateReferences()
	firstOwner := allOwners[0]

	paas := Paas{
		TypeMeta: metav1.TypeMeta{
			Kind:       firstOwner.Kind,
			APIVersion: firstOwner.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: firstOwner.Name,
		},
	}

	for _, ref := range allOwners {
		if ref == firstOwner {
			assert.True(t, paas.IsItMe(ref))
		} else {
			assert.False(t, paas.IsItMe(ref))
		}
	}
	assert.False(t, paas.IsItMe(metav1.OwnerReference{}))
}

func Test_Paas_AmIOwner(t *testing.T) {
	allOwners := generateReferences()
	firstOwner := allOwners[0]

	paas := Paas{
		TypeMeta: metav1.TypeMeta{
			Kind:       firstOwner.Kind,
			APIVersion: firstOwner.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: firstOwner.Name,
		},
	}

	someOwners := []metav1.OwnerReference{
		allOwners[0],
		allOwners[1],
	}
	noOwners := []metav1.OwnerReference{
		allOwners[2],
		allOwners[3],
	}

	empty := []metav1.OwnerReference{}

	assert.True(t, paas.AmIOwner(allOwners))
	assert.True(t, paas.AmIOwner(someOwners))
	assert.False(t, paas.AmIOwner(noOwners))
	assert.False(t, paas.AmIOwner(empty))
}

func Test_Paas_WithoutMe(t *testing.T) {
	allOwners := generateReferences()
	firstOwner := allOwners[0]

	paas := Paas{
		TypeMeta: metav1.TypeMeta{
			Kind:       firstOwner.Kind,
			APIVersion: firstOwner.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: firstOwner.Name,
		},
	}

	someOwners := []metav1.OwnerReference{
		allOwners[0],
		allOwners[1],
	}
	noOwners := []metav1.OwnerReference{
		allOwners[2],
		allOwners[3],
	}

	empty := []metav1.OwnerReference{}

	assert.NotContains(t, paas.WithoutMe(allOwners), allOwners[0])
	assert.Contains(t, paas.WithoutMe(allOwners), allOwners[1])
	assert.NotContains(t, paas.WithoutMe(someOwners), allOwners[0])
	assert.Contains(t, paas.WithoutMe(someOwners), allOwners[1])
	assert.NotContains(t, paas.WithoutMe(noOwners), allOwners[0])
	assert.Contains(t, paas.WithoutMe(noOwners), allOwners[2])
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
				argoCap,
				ssoCap,
				extraCap,
			},
			Capabilities: PaasCapabilities{
				argoCap: PaasCapability{
					Enabled: true,
				},
				grafanaCap: PaasCapability{
					Enabled: true,
				},
				ssoCap: PaasCapability{
					Enabled: false,
				},
			},
		},
	}
	enCapNs := paas.enabledCapNamespaces()
	assert.NotNil(t, enCapNs)
	assert.IsType(t, map[string]bool{}, enCapNs)
	assert.Contains(t, enCapNs, argoCap)
	assert.Contains(t, enCapNs, grafanaCap)
	assert.NotContains(t, enCapNs, ssoCap)
	assert.NotContains(t, enCapNs, extraCap)

	enExNs := paas.extraNamespaces()
	assert.NotNil(t, enExNs)
	assert.IsType(t, map[string]bool{}, enExNs)
	assert.NotContains(t, enExNs, argoCap)
	assert.NotContains(t, enExNs, grafanaCap)
	assert.NotContains(t, enExNs, ssoCap)
	assert.Contains(t, enExNs, extraCap)

	enEnNs := paas.AllEnabledNamespaces()
	assert.NotNil(t, enEnNs)
	assert.IsType(t, map[string]bool{}, enEnNs)
	assert.Contains(t, enEnNs, argoCap)
	assert.Contains(t, enEnNs, grafanaCap)
	assert.NotContains(t, enEnNs, ssoCap)
	assert.Contains(t, enEnNs, extraCap)
}

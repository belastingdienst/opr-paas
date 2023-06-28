package v1alpha1_test

import (
	"sort"
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

var (
	groups = v1alpha1.PaasGroups{
		"cn=test1": v1alpha1.PaasGroup{
			Query: "CN=test2,OU=org_unit,DC=example,DC=org",
			Users: []string{"usr1", "usr2"},
		},
		"cn=test3": v1alpha1.PaasGroup{
			Query: "CN=test4",
			Users: []string{"usr3", "usr2"},
		},
	}
)

func TestPaasGroups_NameFromQuery(t *testing.T) {
	assert.Equal(t, "test2", groups.Key2Name("cn=test1"))
	assert.Equal(t, "", groups.Key2Name("cn=test123"))
	assert.Equal(t, "test4", groups.Key2Name("cn=test3"))
}

func TestPaasGroups_LdapQueries(t *testing.T) {
	ldapGroups := groups.LdapQueries()
	sort.Strings(ldapGroups)
	assert.Equal(t, 2, len(ldapGroups))
	assert.Equal(t, "CN=test2,OU=org_unit,DC=example,DC=org", ldapGroups[0])
	assert.Equal(t, "CN=test4", ldapGroups[1])
}

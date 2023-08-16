package v1alpha1_test

import (
	"sort"
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
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

func TestPaasGroups_QuotaWithDefaults(t *testing.T) {
	testQuotas := map[string]string{
		"limits.cpu":      "3",
		"limits.memory":   "6Gi",
		"requests.cpu":    "800m",
		"requests.memory": "4Gi",
	}
	defaultQuotas := map[string]string{
		"limits.cpu":    "2",
		"limits.memory": "5Gi",
		"requests.cpu":  "700m",
	}
	quotas := make(v1alpha1.PaasQuotas)
	for key, value := range testQuotas {
		quotas[corev1.ResourceName(key)] = resourcev1.MustParse(value)
	}
	defaultedQuotas := quotas.QuotaWithDefaults(defaultQuotas)
	for key, value := range defaultedQuotas {
		if original, exists := quotas[key]; exists {
			assert.Equal(t, original, value)
		}
	}
	assert.Equal(t, defaultedQuotas["requests.memory"],
		resourcev1.MustParse("4Gi"))
	assert.NotEqual(t, defaultedQuotas["requests.cpu"],
		resourcev1.MustParse("700m"))
}

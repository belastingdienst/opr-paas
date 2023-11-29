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
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testGroups = PaasGroups{
		"cn=test1": PaasGroup{
			Query: "CN=test2,OU=org_unit,DC=example,DC=org",
			Users: []string{"usr1", "usr2"},
		},
		"cn=test3": PaasGroup{
			Query: "CN=test4",
			Users: []string{"usr3", "usr2"},
		},
	}
)

func TestPaasGroups_NameFromQuery(t *testing.T) {
	assert.Equal(t, "test2", testGroups.Key2Name("cn=test1"))
	assert.Equal(t, "", testGroups.Key2Name("cn=test123"))
	assert.Equal(t, "test4", testGroups.Key2Name("cn=test3"))
}

func TestPaasGroups_LdapQueries(t *testing.T) {
	ldapGroups := testGroups.LdapQueries()
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
	quotas := make(PaasQuotas)
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

func TestPaasGroups_Namespaces(t *testing.T) {
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
				ArgoCD: PaasArgoCD{
					Enabled: true,
				},
				Grafana: PaasGrafana{
					Enabled: true,
				},
			},
		},
	}
	enCapNs := paas.enabledCapNamespaces()
	assert.Contains(t, enCapNs, "argocd")
	assert.Contains(t, enCapNs, "grafana")
	assert.NotContains(t, enCapNs, "sso")
	assert.NotContains(t, enCapNs, "extra")

	enExNs := paas.extraNamespaces()
	assert.NotContains(t, enExNs, "argocd")
	assert.NotContains(t, enExNs, "grafana")
	assert.NotContains(t, enExNs, "sso")
	assert.Contains(t, enExNs, "extra")

	enEnNs := paas.AllEnabledNamespaces()
	assert.Contains(t, enEnNs, "argocd")
	assert.Contains(t, enEnNs, "grafana")
	assert.NotContains(t, enEnNs, "sso")
	assert.Contains(t, enEnNs, "extra")
}

package quota_test

import (
	"testing"

	paas_quota "github.com/belastingdienst/opr-paas/internal/quota"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
)

func TestPaasQuotas_QuotaWithDefaults(t *testing.T) {
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
	quotas := make(paas_quota.Quotas)
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

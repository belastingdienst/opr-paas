package quota_test

import (
	"testing"

	paasquota "github.com/belastingdienst/opr-paas/v3/internal/quota"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
)

func TestPaasQuotas_QuotaWithDefaults(t *testing.T) {
	testQuotas := map[corev1.ResourceName]resourcev1.Quantity{
		"limits.cpu":      resourcev1.MustParse("3"),
		"limits.memory":   resourcev1.MustParse("6Gi"),
		"requests.cpu":    resourcev1.MustParse("800m"),
		"requests.memory": resourcev1.MustParse("4Gi"),
	}
	defaultQuotas := map[corev1.ResourceName]resourcev1.Quantity{
		"limits.cpu":    resourcev1.MustParse("2"),
		"limits.memory": resourcev1.MustParse("5Gi"),
		"requests.cpu":  resourcev1.MustParse("700m"),
	}
	quotas := make(paasquota.Quota)
	for key, value := range testQuotas {
		quotas[key] = value
	}
	defaultedQuotas := quotas.MergeWith(defaultQuotas)
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

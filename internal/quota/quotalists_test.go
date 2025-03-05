package quota_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	paas_quota "github.com/belastingdienst/opr-paas/internal/quota"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	kiB            int64 = 1024
	MiB                  = kiB * kiB
	GiB                  = MiB * kiB
	quotaCPUKey          = "cpu"
	quotaMemoryKey       = "memory"
	quotaBlockKey        = "block"
	quotaSharedKey       = "shared"
)

var (
	testQuotas = []map[corev1.ResourceName]resource.Quantity{
		{
			quotaCPUKey:    resource.MustParse("3"),
			quotaMemoryKey: resource.MustParse("6Gi"),
			quotaBlockKey:  resource.MustParse("100Gi"),
			quotaSharedKey: resource.MustParse("100Gi"),
		},
		{
			quotaCPUKey:    resource.MustParse("6"),
			quotaMemoryKey: resource.MustParse("12Gi"),
			quotaBlockKey:  resource.MustParse("100Gi"),
		},
		{
			quotaCPUKey:    resource.MustParse("3"),
			quotaMemoryKey: resource.MustParse("12Gi"),
			quotaBlockKey:  resource.MustParse("100Gi"),
		},
	}
	sum_cpu            int64   = 12000
	sum_memory         int64   = 30 * GiB
	min_cpu            int64   = 3000
	min_memory         int64   = 6 * GiB
	max_cpu            int64   = 6000
	max_memory         int64   = 12 * GiB
	largest_two_cpu    int64   = 9000
	largest_two_memory int64   = 24 * GiB
	ratio              float64 = 0.7 // 70%
	optimal_shared     int64   = 100 * GiB
	optimal_block      int64   = 210 * GiB

	minQuota = map[corev1.ResourceName]resource.Quantity{
		quotaCPUKey: resource.MustParse("10"),
	}

	maxQuota = map[corev1.ResourceName]resource.Quantity{
		quotaMemoryKey: resource.MustParse("9Gi"),
	}
)

func TestPaasQuotaLists_Sum(t *testing.T) {
	quotas := paas_quota.NewQuotaLists()
	for _, vals := range testQuotas {
		quotas.Append(vals)
	}
	sum := quotas.Sum()
	cpu, exists := sum[quotaCPUKey]
	assert.True(t, exists, quotaCPUKey+" should exist in sum")
	assert.Equal(t, resource.DecimalSI, cpu.Format, "CPU Should have DecimalSI format")
	assert.Equal(t, sum_cpu, cpu.MilliValue(), "sum should have 12000 milli cpu")
	mem, exists := sum[quotaMemoryKey]
	assert.True(t, exists, quotaMemoryKey+" should exist in sum")
	assert.Equal(t, resource.BinarySI, mem.Format, "Memory should have BinarySI format")
	assert.Equal(t, sum_memory, mem.Value(), "sum should have sum 30 GiB memory")
}

func TestPaasQuotaLists_Min(t *testing.T) {
	quotas := paas_quota.NewQuotaLists()
	for _, vals := range testQuotas {
		quotas.Append(vals)
	}
	minimum := quotas.Min()
	cpu, exists := minimum[quotaCPUKey]
	assert.True(t, exists, quotaCPUKey+" should exist in min")
	assert.Equal(t, resource.DecimalSI, cpu.Format)
	assert.Equal(t, min_cpu, cpu.MilliValue())
	mem, exists := minimum[quotaMemoryKey]
	assert.True(t, exists, quotaMemoryKey+" should exist in min")
	assert.Equal(t, resource.BinarySI, mem.Format)
	assert.Equal(t, min_memory, mem.Value())
}

func TestPaasQuotaLists_Max(t *testing.T) {
	quotas := paas_quota.NewQuotaLists()
	for _, vals := range testQuotas {
		quotas.Append(vals)
	}
	maximum := quotas.Max()
	cpu, exists := maximum[quotaCPUKey]
	assert.True(t, exists, quotaCPUKey+" should exist in max")
	assert.Equal(t, resource.DecimalSI, cpu.Format)
	assert.Equal(t, max_cpu, cpu.MilliValue())
	mem, exists := maximum[quotaMemoryKey]
	assert.True(t, exists)
	assert.Equal(t, resource.BinarySI, mem.Format)
	assert.Equal(t, max_memory, mem.Value())
}

func TestPaasQuotaLists_LargestTwo(t *testing.T) {
	quotas := paas_quota.NewQuotaLists()
	for _, vals := range testQuotas {
		quotas.Append(vals)
	}
	lt := quotas.LargestTwo()
	cpu, exists := lt[quotaCPUKey]
	assert.True(t, exists, quotaCPUKey+" should exist in max")
	assert.Equal(t, resource.DecimalSI, cpu.Format)
	assert.Equal(t, largest_two_cpu, cpu.MilliValue())
	mem, exists := lt[quotaMemoryKey]
	assert.True(t, exists, quotaMemoryKey+" should exist in max")
	assert.Equal(t, resource.BinarySI, mem.Format)
	assert.Equal(t, largest_two_memory, mem.Value())
}

func TestPaasQuotaLists_OptimalValues(t *testing.T) {
	quotas := paas_quota.NewQuotaLists()
	for _, vals := range testQuotas {
		quotas.Append(vals)
	}
	minimum := minQuota
	maximum := maxQuota
	optimal := quotas.OptimalValues(
		ratio,
		minimum,
		maximum,
	)
	cpu := optimal[quotaCPUKey]
	min_cpu := minimum[quotaCPUKey]
	assert.Equal(t, min_cpu.Value(), cpu.Value())
	mem := optimal[quotaMemoryKey]
	max_mem := maximum[quotaMemoryKey]
	assert.Equal(t, max_mem.Value(), mem.Value())
	block := optimal[quotaBlockKey]
	assert.Equal(t, optimal_block, block.Value())
	shared := optimal[quotaSharedKey]
	assert.Equal(t, optimal_shared, shared.Value())
}

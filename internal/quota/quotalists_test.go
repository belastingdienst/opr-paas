package quota_test

import (
	"testing"

	paas_quota "github.com/belastingdienst/opr-paas/internal/quota"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	kiB int64 = 1024
	MiB       = kiB * kiB
	GiB       = MiB * kiB
)

var (
	testQuotas = []map[string]string{
		{
			"cpu":    "3",
			"memory": "6Gi",
			"block":  "100Gi",
			"shared": "100Gi",
		},
		{
			"cpu":    "6",
			"memory": "12Gi",
			"block":  "100Gi",
		},
		{
			"cpu":    "3",
			"memory": "12Gi",
			"block":  "100Gi",
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
	ratio              float64 = 0.7
	optimal_shared     int64   = 100 * GiB
	optimal_block      int64   = 210 * GiB

	minQuota = map[string]string{
		"cpu": "10",
	}

	maxQuota = map[string]string{
		"memory": "9Gi",
	}
)

func TestPaasQuotaLists_Sum(t *testing.T) {
	quotas := paas_quota.NewQuotaLists()
	for _, vals := range testQuotas {
		quotas.Append(paas_quota.NewQuota(vals))
	}
	sum := quotas.Sum()
	cpu, exists := sum["cpu"]
	assert.True(t, exists, "cpu should exist in sum")
	assert.Equal(t, resource.DecimalSI, cpu.Format, "CPU Should have DecimalSI format")
	assert.Equal(t, sum_cpu, cpu.MilliValue(), "sum should have 12000 milli cpu")
	mem, exists := sum["memory"]
	assert.True(t, exists, "memory should exist in sum")
	assert.Equal(t, resource.BinarySI, mem.Format, "Memory should have BinarySI format")
	assert.Equal(t, sum_memory, mem.Value(), "sum should have sum 30 GiB memory")
}

func TestPaasQuotaLists_Min(t *testing.T) {
	quotas := paas_quota.NewQuotaLists()
	for _, vals := range testQuotas {
		quotas.Append(paas_quota.NewQuota(vals))
	}
	min := quotas.Min()
	cpu, exists := min["cpu"]
	assert.True(t, exists, "cpu should exist in min")
	assert.Equal(t, resource.DecimalSI, cpu.Format)
	assert.Equal(t, min_cpu, cpu.MilliValue())
	mem, exists := min["memory"]
	assert.True(t, exists, "memory should exist in min")
	assert.Equal(t, resource.BinarySI, mem.Format)
	assert.Equal(t, min_memory, mem.Value())
}

func TestPaasQuotaLists_Max(t *testing.T) {
	quotas := paas_quota.NewQuotaLists()
	for _, vals := range testQuotas {
		quotas.Append(paas_quota.NewQuota(vals))
	}
	max := quotas.Max()
	cpu, exists := max["cpu"]
	assert.True(t, exists, "cpu should exist in max")
	assert.Equal(t, resource.DecimalSI, cpu.Format)
	assert.Equal(t, max_cpu, cpu.MilliValue())
	mem, exists := max["memory"]
	assert.True(t, exists)
	assert.Equal(t, resource.BinarySI, mem.Format)
	assert.Equal(t, max_memory, mem.Value())
}

func TestPaasQuotaLists_LargestTwo(t *testing.T) {
	quotas := paas_quota.NewQuotaLists()
	for _, vals := range testQuotas {
		quotas.Append(paas_quota.NewQuota(vals))
	}
	lt := quotas.LargestTwo()
	cpu, exists := lt["cpu"]
	assert.True(t, exists, "cpu should exist in max")
	assert.Equal(t, resource.DecimalSI, cpu.Format)
	assert.Equal(t, largest_two_cpu, cpu.MilliValue())
	mem, exists := lt["memory"]
	assert.True(t, exists, "memory should exist in max")
	assert.Equal(t, resource.BinarySI, mem.Format)
	assert.Equal(t, largest_two_memory, mem.Value())
}

func TestPaasQuotaLists_OptimalValues(t *testing.T) {
	quotas := paas_quota.NewQuotaLists()
	for _, vals := range testQuotas {
		quotas.Append(paas_quota.NewQuota(vals))
	}
	min := paas_quota.NewQuota(minQuota)
	max := paas_quota.NewQuota(maxQuota)
	optimal := quotas.OptimalValues(
		ratio,
		min,
		max,
	)
	cpu := optimal["cpu"]
	min_cpu := min["cpu"]
	assert.Equal(t, min_cpu.Value(), cpu.Value())
	mem := optimal["memory"]
	max_mem := max["memory"]
	assert.Equal(t, max_mem.Value(), mem.Value())
	block := optimal["block"]
	assert.Equal(t, optimal_block, block.Value())
	shared := optimal["shared"]
	assert.Equal(t, optimal_shared, shared.Value())
}

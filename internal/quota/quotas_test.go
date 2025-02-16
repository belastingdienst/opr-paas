package quota_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

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
	testQuotas = []map[corev1.ResourceName]resource.Quantity{
		{
			"cpu":    resource.MustParse("3"),
			"memory": resource.MustParse("6Gi"),
			"block":  resource.MustParse("100Gi"),
			"shared": resource.MustParse("100Gi"),
		},
		{
			"cpu":    resource.MustParse("6"),
			"memory": resource.MustParse("12Gi"),
			"block":  resource.MustParse("100Gi"),
		},
		{
			"cpu":    resource.MustParse("3"),
			"memory": resource.MustParse("12Gi"),
			"block":  resource.MustParse("100Gi"),
		},
	}
	sumCPU           int64 = 12000
	sumMemory              = 30 * GiB
	minCPU           int64 = 3000
	minMemory              = 6 * GiB
	maxCPU           int64 = 6000
	maxMemory              = 12 * GiB
	largestTwoCPU    int64 = 9000
	largestTwoMemory       = 24 * GiB
	ratio                  = 0.7 // 70%
	optimalShared          = 100 * GiB
	optimalBlock           = 210 * GiB

	minQuota = map[corev1.ResourceName]resource.Quantity{
		"cpu": resource.MustParse("10"),
	}

	maxQuota = map[corev1.ResourceName]resource.Quantity{
		"memory": resource.MustParse("9Gi"),
	}
)

func TestPaasQuotas_Sum(t *testing.T) {
	quotas := paas_quota.NewQuotas()
	for _, vals := range testQuotas {
		quotas.Append(vals)
	}
	sum := quotas.Sum()
	cpu, exists := sum["cpu"]
	assert.True(t, exists, "cpu should exist in sum")
	assert.Equal(t, resource.DecimalSI, cpu.Format, "CPU Should have DecimalSI format")
	assert.Equal(t, sumCPU, cpu.MilliValue(), "sum should have 12000 milli cpu")
	mem, exists := sum["memory"]
	assert.True(t, exists, "memory should exist in sum")
	assert.Equal(t, resource.BinarySI, mem.Format, "Memory should have BinarySI format")
	assert.Equal(t, sumMemory, mem.Value(), "sum should have sum 30 GiB memory")
}

func TestPaasQuotas_Min(t *testing.T) {
	quotas := paas_quota.NewQuotas()
	for _, vals := range testQuotas {
		quotas.Append(vals)
	}
	min := quotas.Min()
	cpu, exists := min["cpu"]
	assert.True(t, exists, "cpu should exist in min")
	assert.Equal(t, resource.DecimalSI, cpu.Format)
	assert.Equal(t, minCPU, cpu.MilliValue())
	mem, exists := min["memory"]
	assert.True(t, exists, "memory should exist in min")
	assert.Equal(t, resource.BinarySI, mem.Format)
	assert.Equal(t, minMemory, mem.Value())
}

func TestPaasQuotas_Max(t *testing.T) {
	quotas := paas_quota.NewQuotas()
	for _, vals := range testQuotas {
		quotas.Append(vals)
	}
	max := quotas.Max()
	cpu, exists := max["cpu"]
	assert.True(t, exists, "cpu should exist in max")
	assert.Equal(t, resource.DecimalSI, cpu.Format)
	assert.Equal(t, maxCPU, cpu.MilliValue())
	mem, exists := max["memory"]
	assert.True(t, exists)
	assert.Equal(t, resource.BinarySI, mem.Format)
	assert.Equal(t, maxMemory, mem.Value())
}

func TestPaasQuotas_LargestTwo(t *testing.T) {
	quotas := paas_quota.NewQuotas()
	for _, vals := range testQuotas {
		quotas.Append(vals)
	}
	lt := quotas.LargestTwo()
	cpu, exists := lt["cpu"]
	assert.True(t, exists, "cpu should exist in max")
	assert.Equal(t, resource.DecimalSI, cpu.Format)
	assert.Equal(t, largestTwoCPU, cpu.MilliValue())
	mem, exists := lt["memory"]
	assert.True(t, exists, "memory should exist in max")
	assert.Equal(t, resource.BinarySI, mem.Format)
	assert.Equal(t, largestTwoMemory, mem.Value())
}

func TestPaasQuotas_OptimalValues(t *testing.T) {
	quotas := paas_quota.NewQuotas()
	for _, vals := range testQuotas {
		quotas.Append(vals)
	}
	min := minQuota
	max := maxQuota
	optimal := quotas.OptimalValues(
		ratio,
		min,
		max,
	)
	cpu := optimal["cpu"]
	minCPU := min["cpu"]
	assert.Equal(t, minCPU.Value(), cpu.Value())
	mem := optimal["memory"]
	maxMemory := max["memory"]
	assert.Equal(t, maxMemory.Value(), mem.Value())
	block := optimal["block"]
	assert.Equal(t, optimalBlock, block.Value())
	shared := optimal["shared"]
	assert.Equal(t, optimalShared, shared.Value())
}

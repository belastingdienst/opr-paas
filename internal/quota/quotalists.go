package quota

/*
QuotaLists is a type which is especially designed to collect and summarize information about quota.
Quota are basically maps with resource names as key and quantities as values, e.a.:
cpu: 100m
memory: 1GiB
storage.nfs: 100GiB

Multiple Paas'es could have multiple CLusterwide quota's, each with it's own list of items.
QuotaLists are meant to bring them (across Paas'es an even across capabilities) in maps of lists of quantities.
After collecting the info, QuotaLists can summarize (e.a. min, max, sum all values, sum largest two values, etc.).
And QuotaLists can combine these summarizing techniques to calculate the optimal value for eacht quotum (key, value pair).
*/

import (
	"sort"

	v1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
)

// map[paas][quotatype]value
// type QuotaLists map[string]Quotas
type QuotaLists struct {
	list map[v1.ResourceName][]resourcev1.Quantity
}

func NewQuotaLists() QuotaLists {
	return QuotaLists{
		list: make(map[v1.ResourceName][]resourcev1.Quantity),
	}
}

func (pcr *QuotaLists) Append(quotas Quotas) {
	for key, value := range quotas {
		if values, exists := pcr.list[key]; exists {
			pcr.list[key] = append(values, value)
		} else {
			pcr.list[key] = []resourcev1.Quantity{value}
		}
	}
}

func (pcr QuotaLists) Sum() Quotas {
	quotaResources := make(Quotas)
	for key, values := range pcr.list {
		var newValue resourcev1.Quantity
		for _, value := range values {
			newValue.Add(value)
		}
		quotaResources[key] = newValue
	}
	return quotaResources
}

func (pcr QuotaLists) LargestTwo() Quotas {
	quotaResources := make(Quotas)
	for key, values := range pcr.list {
		if len(values) == 1 {
			quotaResources[key] = values[0]
		} else if len(values) > 1 {
			sort.Slice(values, func(i, j int) bool { return values[i].Value() > values[j].Value() })
			value := values[0]
			value.Add(values[1])
			quotaResources[key] = value
		}
	}
	return quotaResources
}

func (pcr QuotaLists) Max() Quotas {
	quotaResources := make(Quotas)
	for key, values := range pcr.list {
		if len(values) < 1 {
			quotaResources[key] = resourcev1.MustParse("0")
			continue
		}
		sort.Slice(values, func(i, j int) bool { return values[i].Value() > values[j].Value() })
		quotaResources[key] = values[0]
	}
	return quotaResources
}

func (pcr QuotaLists) Min() Quotas {
	quotaResources := make(Quotas)
	for key, values := range pcr.list {
		if len(values) < 1 {
			quotaResources[key] = resourcev1.MustParse("0")
			continue
		}
		sort.Slice(values, func(i, j int) bool { return values[i].Value() < values[j].Value() })
		quotaResources[key] = values[0]
	}
	return quotaResources
}

func (pcr QuotaLists) OptimalValues(ratio float64, minQuotas Quotas, maxQuotas Quotas) Quotas {
	// Calculate resources with 3 different approaches and select largest value
	approaches := NewQuotaLists()
	approaches.Append(pcr.Sum().Resized(ratio))
	approaches.Append(pcr.LargestTwo())
	approaches.Append(minQuotas)
	// Cap with max values from config
	capped := NewQuotaLists()
	capped.Append(approaches.Max())
	capped.Append(maxQuotas)
	// return optimal values as derived from config and values
	return capped.Min()
}

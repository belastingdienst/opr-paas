package quota

/*
Quotas is a type which is especially designed to collect and summarize information about quota.
Quota are basically maps with resource names as key and quantities as values, e.a.:
cpu: 100m
memory: 1GiB
storage.nfs: 100GiB

Multiple Paas'es could have multiple CLusterwide quota's, each with its own list of items.
Quotas are meant to bring them (across Paas'es an even across capabilities) in maps of lists of quantities.
After collecting the info, Quotas can summarize (e.a. min, max, sum all values, sum largest two values, etc.).
Quotas can combine these summarizing techniques to calculate the optimal value for each quotum (key, value pair).
*/

import (
	"sort"

	k8sv1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
)

// Quotas is a struct to merge multiple resources.
// For every resource name, it hols a list of all quantities that has been appended
type Quotas struct {
	list map[k8sv1.ResourceName][]resourcev1.Quantity
}

// NewQuotas can be used to instantiate a new block (running make on the internal list)
func NewQuotas() Quotas {
	return Quotas{
		list: make(map[k8sv1.ResourceName][]resourcev1.Quantity),
	}
}

// Append can be used to append quotas.
// For resource names that had previously been fed, it appends to the existing list.
// For new resource names is creating a new list.
func (pcr *Quotas) Append(quotas Quota) {
	for key, value := range quotas {
		if values, exists := pcr.list[key]; exists {
			pcr.list[key] = append(values, value)
		} else {
			pcr.list[key] = []resourcev1.Quantity{value}
		}
	}
}

// Sum will return a quota with for every resource name the sum of the list of quota's previously appended
func (pcr Quotas) Sum() Quota {
	quotaResources := make(Quota)
	for key, values := range pcr.list {
		var newValue resourcev1.Quantity
		for _, value := range values {
			newValue.Add(value)
		}
		quotaResources[key] = newValue
	}
	return quotaResources
}

// LargestTwo returns a Quota with the sum of the largest two Quotas for each resource name that was previously
// appended.
func (pcr Quotas) LargestTwo() Quota {
	quotaResources := make(Quota)
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

// Max returns a Quota with the largest Quota for each resource name that was previously appended.
func (pcr Quotas) Max() Quota {
	quotaResources := make(Quota)
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

// Min returns a Quota with the smallest Quota for each resource name that was previously appended.
func (pcr Quotas) Min() Quota {
	quotaResources := make(Quota)
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

// OptimalValues calculates optimal values using multiple angles largest of (minimum, sum*ratio, largest two),
// capped by max
func (pcr Quotas) OptimalValues(ratio float64, minQuotas Quota, maxQuotas Quota) Quota {
	// Calculate resources with 3 different approaches and select largest value
	approaches := NewQuotas()
	approaches.Append(pcr.Sum().Resized(ratio))
	approaches.Append(pcr.LargestTwo())
	approaches.Append(minQuotas)
	// Cap with max values from config
	capped := NewQuotas()
	capped.Append(approaches.Max())
	capped.Append(maxQuotas)
	// return optimal values as derived from config and values
	return capped.Min()
}

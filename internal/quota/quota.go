package quota

import (
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
)

// Quota holds a map of resource quantities
// The main reason for having this as a separate type is to add methods
type Quota map[corev1.ResourceName]resourcev1.Quantity

// MergeWith can be used to merge 2 Quota blocks
func (pq Quota) MergeWith(targetQuota map[corev1.ResourceName]resourcev1.Quantity) (q Quota) {
	q = make(Quota)
	for key, value := range targetQuota {
		q[key] = value
	}
	for key, value := range pq {
		q[key] = value
	}
	return q
}

// Resized can be used to scale a quota block
func (pq Quota) Resized(scale float64) (q Quota) {
	q = make(Quota)
	for key, value := range pq {
		resized := value.AsApproximateFloat64() * scale
		q[key] = *(resourcev1.NewQuantity(int64(resized), value.Format))
	}
	return q
}

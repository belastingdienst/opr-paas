package quota

import (
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
)

type Quotas map[corev1.ResourceName]resourcev1.Quantity

func (pq Quotas) QuotaWithDefaults(defaults map[string]string) (q Quotas) {
	q = make(Quotas)
	for key, value := range defaults {
		q[corev1.ResourceName(key)] = resourcev1.MustParse(value)
	}
	for key, value := range pq {
		q[key] = value
	}
	return q
}

func (pq Quotas) Resized(scale float64) (q Quotas) {
	q = make(Quotas)
	for key, value := range pq {
		resized := value.AsApproximateFloat64() * scale
		q[corev1.ResourceName(key)] = *(resourcev1.NewQuantity(int64(resized), value.Format))
	}
	return q
}

func NewQuota(defaults map[string]string) (q Quotas) {
	q = make(Quotas)
	for key, value := range defaults {
		q[corev1.ResourceName(key)] = resourcev1.MustParse(value)
	}
	return q
}

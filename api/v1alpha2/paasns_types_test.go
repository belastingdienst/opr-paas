package v1alpha2

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PaasnsTypes", func() {
	const (
		paasName   = "mypaas"
		paasNsName = "mypaasns"
		nsName     = paasName + "-" + paasNsName
	)
	var (
		paasNsLables = map[string]string{
			instanceLabel: paasName,
			"some-label":  "some-value",
			"other-label": "other-value",
		}
		paasns = PaasNS{
			ObjectMeta: metav1.ObjectMeta{
				Name:      paasNsName,
				Namespace: paasName,
				Labels:    paasNsLables,
			},
		}
		ownedObject = corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsName,
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: GroupVersion.Identifier(), Kind: "PaasNS", Name: paasNsName, Namespace: paasName},
				},
			},
		}
	)

	BeforeAll(func() {
	})
	When("cloning labels", func() {
		Context("on a paasns with labels", Ordered, func() {
			labels := paasns.ClonedLabels()
			It("should not return the "+instanceLabel+" label", func() {
				Expect(paasNsLables).To(HaveKey(instanceLabel))
				Expect(labels).NotTo(HaveKey(instanceLabel))
			})
			It("should return all other labels", func() {
				for key, value := range paasNsLables {
					Expect(labels).To(HaveKeyWithValue(key, value))
				}
			})
		})
	})
	When("checking paasns ownership", func() {
		Context("on a resource", Ordered, func() {
			It("should return when ownership is properly configured", func() {
			})
		})
	})
})

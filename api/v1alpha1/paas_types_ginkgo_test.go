/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/v3/pkg/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Namespace Validation", func() {
	const resourceName = "test-paas"

	Context("Valid namespacesnames", func() {
		It("should accept a valid namespace", func() {
			validNamespace := []string{"valid-namespace-example"}
			paas := &Paas{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
				Spec: PaasSpec{
					Quota:      make(quota.Quota),
					Requestor:  "valid-requestor",
					Namespaces: validNamespace,
				},
			}

			err := k8sClient.Create(context.TODO(), paas)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(context.Background(), paas)).To(Succeed())
		})
	})

	Context("Invalid namespacenames", func() {
		DescribeTable("should reject invalid names",
			func(namespaces []string) {
				paas := &Paas{
					ObjectMeta: metav1.ObjectMeta{
						Name: resourceName,
					},
					Spec: PaasSpec{
						Quota:      make(quota.Quota),
						Requestor:  "valid-requestor",
						Namespaces: namespaces,
					},
				}

				err := k8sClient.Create(context.TODO(), paas)
				Expect(err).To(HaveOccurred()) // Expect validation to fail
			},
			Entry("starts with a hyphen", []string{"-invalid.com"}),
			Entry("contains a dot", []string{"valid", "invalid.com"}),
			Entry("ends with a hyphen", []string{"invalid-"}),
			Entry("contains uppercase letters", []string{"Invalid-com"}),
			Entry("contains special characters", []string{"invalid!name.com"}),
			Entry("exceeds max length", []string{fmt.Sprintf("%s-com", string(make([]byte, 254)))}),
		)
	})
})

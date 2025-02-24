/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/internal/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Namespace Validation", func() {
	const resourceName = "test-paas"

	Context("Valid Hostname", func() {
		It("should accept a valid hostname", func() {
			validNamespace := []string{"valid-hostname.example.com"}
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

	Context("Invalid Hostnames", func() {
		DescribeTable("should reject invalid hostnames",
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
			Entry("starts with a dot", []string{"valid", ".invalid.com"}),
			Entry("ends with a hyphen", []string{"invalid-.com-"}),
			Entry("contains uppercase letters", []string{"Invalid.com"}),
			Entry("contains special characters", []string{"invalid!name.com"}),
			Entry("exceeds max length", []string{fmt.Sprintf("%s.com", string(make([]byte, 254)))}),
		)
	})
})

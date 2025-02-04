/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PaasConfig Webhook", func() {
	var (
		obj       *v1alpha1.PaasConfig
		oldObj    *v1alpha1.PaasConfig
		validator PaasConfigCustomValidator
	)

	BeforeEach(func() {
		obj = &v1alpha1.PaasConfig{}
		oldObj = &v1alpha1.PaasConfig{}
		validator = PaasConfigCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
	})

	Context("When creating or updating PaasConfig under Validating Webhook", func() {
		// TODO (portly-halicore-76): Add logic for validating webhooks
		// Example:
		It("Should deny creation", func() {
			By("simulating an invalid creation scenario")
			Expect(validator.ValidateCreate(ctx, obj)).Error().ToNot(HaveOccurred())
		})

		//It("Should admit creation if all required fields are present", func() {
		//	By("simulating an invalid creation scenario")
		//	Expect(validator.ValidateCreate(ctx, obj)).To(BeNil())
		//})
		//
		// It("Should validate updates correctly", func() {
		//     By("simulating a valid update scenario")
		//     oldObj.SomeRequiredField = "updated_value"
		//     obj.SomeRequiredField = "updated_value"
		//     Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil())
		// })
	})
})

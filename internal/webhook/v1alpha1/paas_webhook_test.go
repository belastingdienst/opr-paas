/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Paas Webhook", func() {
	var (
		obj       *v1alpha1.Paas
		oldObj    *v1alpha1.Paas
		validator PaasCustomValidator
	)

	BeforeEach(func() {
		obj = &v1alpha1.Paas{}
		oldObj = &v1alpha1.Paas{}
		validator = PaasCustomValidator{k8sClient}
		conf := v1alpha1.PaasConfig{
			Spec: v1alpha1.PaasConfigSpec{
				Capabilities: v1alpha1.ConfigCapabilities{},
			},
		}

		config.SetConfig(conf)
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
	})

	Context("When creating a Paas under Validating Webhook", func() {
		It("Should deny creation when a capability is set that is not configured", func() {
			obj = &v1alpha1.Paas{
				Spec: v1alpha1.PaasSpec{
					Capabilities: v1alpha1.PaasCapabilities{"foo": v1alpha1.PaasCapability{}},
				},
			}

			Expect(validator.ValidateCreate(ctx, obj)).Error().To(MatchError("capability foo not configured"))
		})

		//It("Should admit creation if all required fields are present", func() {
		//	By("simulating an invalid creation scenario")
		//	Expect(validator.ValidateCreate(ctx, obj)).To(BeNil())
		//})
	})

	Context("When updating a Paas under Validating Webhook", func() {
		It("Should deny creation when a capability is set that is not configured", func() {
			oldObj = &v1alpha1.Paas{Spec: v1alpha1.PaasSpec{
				Capabilities: v1alpha1.PaasCapabilities{},
			}}
			obj = &v1alpha1.Paas{Spec: v1alpha1.PaasSpec{
				Capabilities: v1alpha1.PaasCapabilities{"foo": v1alpha1.PaasCapability{}},
			}}

			Expect(validator.ValidateCreate(ctx, oldObj)).Error().ToNot(HaveOccurred())
			Expect(validator.ValidateUpdate(ctx, oldObj, obj)).Error().To(MatchError("capability foo not configured"))
		})

		// It("Should validate updates correctly", func() {
		//     By("simulating a valid update scenario")
		//     oldObj.SomeRequiredField = "updated_value"
		//     obj.SomeRequiredField = "updated_value"
		//     Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil())
		// })
	})
})

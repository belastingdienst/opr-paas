/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("PaasConfig Webhook", func() {
	var (
		obj       *v1alpha1.PaasConfig
		oldObj    *v1alpha1.PaasConfig
		validator PaasConfigCustomValidator
		ctx       context.Context
		scheme    *runtime.Scheme
		cl        client.Client
	)

	BeforeEach(func() {
		existing := &v1alpha1.PaasConfig{}
		scheme = runtime.NewScheme()
		Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())

		// Create a fake client that already has the existing PaasConfig
		cl = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(existing).
			Build()

		obj = &v1alpha1.PaasConfig{}
		oldObj = &v1alpha1.PaasConfig{}
		validator = PaasConfigCustomValidator{client: cl}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
	})

	When("creating a PaasConfig under Validating Webhook", func() {
		Context("and a PaasConfig resource already exists", func() {
			It("should deny creation", func() {
				obj = &v1alpha1.PaasConfig{}

				Expect(validator.ValidateCreate(ctx, obj)).Error().To(HaveOccurred())
			})
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

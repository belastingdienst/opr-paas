/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

//revive:disable:dot-imports

import (
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Creating a PaasConfig", func() {
	var (
		obj       *v1alpha1.PaasConfig
		oldObj    *v1alpha1.PaasConfig
		validator PaasConfigCustomValidator
		scheme    *runtime.Scheme
		cl        client.Client
	)

	BeforeEach(func() {
		obj = &v1alpha1.PaasConfig{}
		oldObj = &v1alpha1.PaasConfig{}
		validator = PaasConfigCustomValidator{client: k8sClient}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
	})

	When("creating a PaasConfig under Validating Webhook", func() {
		Context("and a PaasConfig resource already exists", func() {
			It("should deny creation", func() {
				existing := &v1alpha1.PaasConfig{}
				scheme = runtime.NewScheme()
				Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())

				// Create a fake client that already has the existing PaasConfig
				cl = fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(existing).
					Build()

				validator = PaasConfigCustomValidator{client: cl}
				obj = &v1alpha1.PaasConfig{}

				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(warn, err).Error().To(HaveOccurred())
				fmt.Printf("%s", err.Error())
				Expect(err.Error()).To(Equal("[]: Forbidden: another PaasConfig resource already exists"))
			})
		})

		Context("and no PaasConfig already exists", func() {
			Context("and the new PaasConfig does not have one or more required fields", func() {
				It("should deny creation", func() {
					warn, err := validator.ValidateCreate(ctx, obj)
					Expect(warn, err).Error().To(HaveOccurred())
					Expect(err).To(HaveLen(4))
					Expect(err.Error()).To(Equal("[]: Forbidden: another PaasConfig resource already exists"))
				})
			})
		})
	})
})

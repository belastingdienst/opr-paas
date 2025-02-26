/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

// Excuse Ginkgo use from revive errors
//revive:disable:dot-imports

import (
	"errors"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		obj = &v1alpha1.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "newPaasConfig"},
			Spec: v1alpha1.PaasConfigSpec{
				LDAP: v1alpha1.ConfigLdap{
					Host: "some-invalid-hostname",
					Port: 3309,
				},
				ExcludeAppSetName: "Something something",
			},
		}
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
				obj = &v1alpha1.PaasConfig{
					ObjectMeta: metav1.ObjectMeta{Name: "newPaasConfig"},
				}

				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(warn, err).Error().To(HaveOccurred())
				Expect(err.Error()).To(
					//revive:disable-next-line
					Equal(`PaasConfig.cpet.belastingdienst.nl "newPaasConfig" is invalid: spec: Forbidden: another PaasConfig resource already exists`))
			})
		})

		Context("and no PaasConfig already exists", func() {
			Context("and the new PaasConfig does not have one or more required fields", func() {
				It("should deny creation", func() {
					warn, err := validator.ValidateCreate(ctx, obj)
					fmt.Printf("DEBUG - %v", warn)
					Expect(err).Error().To(HaveOccurred())
					Expect(warn).To(HaveLen(1))
					Expect(warn[0]).To(Equal("spec.excludeappsetname: deprecated"))

					var serr *apierrors.StatusError
					Expect(errors.As(err, &serr)).To(BeTrue())

					causes := serr.Status().Details.Causes
					Expect(causes).To(HaveLen(2))
					expectedErrors := []metav1.StatusCause{
						{
							Type:    "FieldValueInvalid",
							Message: `Invalid value: "some-invalid-hostname": invalid host name / ip address`,
							Field:   "spec.LDAP",
						},
						{
							Type: "FieldValueRequired",
							//revive:disable-next-line
							Message: "Required value: DecryptKeysSecret is required and must have both name and namespace",
							Field:   "spec.decryptkeyssecret",
						},
					}
					Expect(causes).To(ConsistOf(expectedErrors))
				})
			})
		})
	})
})

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

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Creating a PaasConfig", Ordered, func() {
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
					Host: "some-invalid-hostname.nl",
					Port: 3309,
				},
				ExcludeAppSetName: "Something something",
				DecryptKeysSecret: v1alpha1.NamespacedName{
					Name:      paasConfigPkSecret,
					Namespace: paasConfigSystem,
				},
				Validations: map[string]map[string]string{
					"paas": {
						"groupNames": "[0-9a-z-]{1,63}",
					},
				},
			},
		}
		oldObj = &v1alpha1.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "newPaasConfig"},
			Spec: v1alpha1.PaasConfigSpec{
				DecryptKeysSecret: v1alpha1.NamespacedName{
					Name:      "keys",
					Namespace: "paas-system-config",
				},
				LDAP: v1alpha1.ConfigLdap{
					Host: "some-other-valid-hostname.nl",
					Port: 3310,
				},
				ExcludeAppSetName: "Another something",
				Validations: map[string]map[string]string{
					"paas": {
						"groupNames": "[0-9A-Za-z-]{1,128}",
					},
				},
			},
		}
		validator = PaasConfigCustomValidator{client: k8sClient}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
	})

	When("creating a PaasConfig", func() {
		Context("with valid definition under valid circumstances", func() {
			It("should succeed", func() {
				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(warn, err).Error().NotTo(HaveOccurred())
			})
		})
		Context("with invalid Validation regular expressions", func() {
			It("should raise an error", func() {
				obj.Spec.Validations["paas"]["groupName"] = ".*)"
				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(warn, err).Error().To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`failed to compile validation regexp for paas.groupName`))
			})
		})
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
					Expect(err).Error().To(HaveOccurred())
					Expect(warn).To(HaveLen(1))
					Expect(warn[0]).To(Equal("spec.excludeappsetname: deprecated"))

					var serr *apierrors.StatusError
					Expect(errors.As(err, &serr)).To(BeTrue())

					causes := serr.Status().Details.Causes
					Expect(causes).To(HaveLen(1))
					expectedErrors := []metav1.StatusCause{
						{
							Type:    "FieldValueInvalid",
							Message: `Invalid value: "some-invalid-hostname": invalid host name / ip address`,
							Field:   "spec.LDAP",
						},
					}
					Expect(causes).To(ConsistOf(expectedErrors))
				})
			})
		})
	})

	When("creating a new PaasConfig", func() {
		Context("having a capability defined with clusterwide=true", func() {
			It("should not check if Min > Def", func() {
				// Ensure correct PaasConfig
				obj.Spec.LDAP.Host = "example.com"

				// Add cap for testing
				obj.Spec.Capabilities = v1alpha1.ConfigCapabilities{
					"HighQuotaCapability": v1alpha1.ConfigCapability{
						AppSet: "high-quota-appset",
						QuotaSettings: v1alpha1.ConfigQuotaSettings{
							Clusterwide: true,
							DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
								corev1.ResourceCPU:    resourcev1.MustParse("5000m"),
								corev1.ResourceMemory: resourcev1.MustParse("1Gi"),
							},
							MinQuotas: map[corev1.ResourceName]resourcev1.Quantity{
								corev1.ResourceCPU:    resourcev1.MustParse("1000m"),
								corev1.ResourceMemory: resourcev1.MustParse("2Gi"),
							},
						},
					},
				}

				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(err).Error().To(Not(HaveOccurred()))
				Expect(warn).To(BeEmpty())
			})
		})
	})
})

var _ = Describe("Updating a PaasConfig", Ordered, func() {
	var (
		obj       *v1alpha1.PaasConfig
		oldObj    *v1alpha1.PaasConfig
		validator PaasConfigCustomValidator
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
				DecryptKeysSecret: v1alpha1.NamespacedName{
					Name:      paasConfigPkSecret,
					Namespace: paasConfigSystem,
				},
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

	When("updating an existing PaasConfig", func() {
		Context("having a capability defined with clusterwide=true", func() {
			It("should not check if Min > Def", func() {
				// Ensure correct PaasConfig
				obj.Spec.LDAP.Host = "example.com"

				// Add cap for testing
				obj.Spec.Capabilities = v1alpha1.ConfigCapabilities{
					"HighQuotaCapability": v1alpha1.ConfigCapability{
						AppSet: "high-quota-appset",
						QuotaSettings: v1alpha1.ConfigQuotaSettings{
							Clusterwide: true,
							DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
								corev1.ResourceCPU:    resourcev1.MustParse("5000m"),
								corev1.ResourceMemory: resourcev1.MustParse("1Gi"),
							},
							MinQuotas: map[corev1.ResourceName]resourcev1.Quantity{
								corev1.ResourceCPU:    resourcev1.MustParse("1000m"),
								corev1.ResourceMemory: resourcev1.MustParse("2Gi"),
							},
						},
					},
				}

				warn, err := validator.ValidateUpdate(ctx, oldObj, obj)
				Expect(err).Error().To(Not(HaveOccurred()))
				Expect(warn).To(BeEmpty())
			})
		})
	})
	When("updating a PaasConfig", func() {
		Context("with valid definition changes under valid circumstances", func() {
			It("should succeed", func() {
				warn, err := validator.ValidateUpdate(ctx, oldObj, obj)
				Expect(warn, err).Error().NotTo(HaveOccurred())
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("with invalid Validation regular expressions", func() {
			It("should raise an error", func() {
				obj.Spec.Validations["paas"]["groupName"] = ".*)"
				warn, err := validator.ValidateUpdate(ctx, oldObj, obj)
				Expect(warn, err).Error().To(HaveOccurred())
				Expect(err.Error()).To(
					ContainSubstring(`failed to compile validation regexp for paas.groupName`))
			})
		})
	})
})

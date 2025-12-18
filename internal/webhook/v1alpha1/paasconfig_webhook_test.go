/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

// Excuse Ginkgo use from revive errors
//revive:disable:dot-imports

import (
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/v4/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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
				DecryptKeysSecret: v1alpha1.NamespacedName{
					Name:      paasConfigPkSecret,
					Namespace: paasConfigSystem,
				},
				Validations: v1alpha1.PaasConfigValidations{
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
				Validations: v1alpha1.PaasConfigValidations{
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
				_, err := validator.ValidateCreate(ctx, obj)
				Expect(err).Error().To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`failed to compile validation regexp for paas.groupName`))
			})
		})
		Context("with deprecated fields", func() {
			It("should raise an error", func() {
				obj.Spec.ExcludeAppSetName = "something something"
				obj.Spec.ArgoPermissions = v1alpha1.ConfigArgoPermissions{
					Header: "something",
				}
				warn, err := validator.ValidateCreate(ctx, obj)

				Expect(err).Error().NotTo(HaveOccurred())
				Expect(warn).To(HaveLen(2))
				Expect(warn).To(ContainElements(
					"spec.argopermissions: deprecated",
					"spec.excludeappsetname: deprecated",
				))
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
		Context("quota name validation", func() {
			var (
				validResourceKeys = []string{
					// "limits.cpu",
					"limits.memory",
					"requests.cpu",
					"requests.memory",
					"requests.storage",
					"thin.storageclass.storage.k8s.io/persistentvolumeclaims",
				}
				validQuotas = map[corev1.ResourceName]resourcev1.Quantity{
					"limits.memory":    resourcev1.MustParse("100M"),
					"requests.cpu":     resourcev1.MustParse("1.1"),
					"requests.memory":  resourcev1.MustParse("100M"),
					"requests.storage": resourcev1.MustParse("10G"),
					"thin.storageclass.storage.k8s.io/persistentvolumeclaims": resourcev1.MustParse("10G"),
				}
				validation    = fmt.Sprintf("(%s)", strings.Join(validResourceKeys, "|"))
				invalidQuotas = map[corev1.ResourceName]resourcev1.Quantity{
					"limits.cpu": resourcev1.MustParse("1.1"),
				}
			)
			It("should allow names that meet re", func() {
				obj.Spec.Validations["paas"]["allowedQuotas"] = validation
				obj.Spec.Capabilities = v1alpha1.ConfigCapabilities{
					"quota": v1alpha1.ConfigCapability{
						QuotaSettings: v1alpha1.ConfigQuotaSettings{
							DefQuota:  validQuotas,
							MinQuotas: validQuotas,
							MaxQuotas: validQuotas,
						},
					},
				}
				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(warn, err).Error().NotTo(HaveOccurred())
			})
			It("should deny names that do not meet re", func() {
				obj.Spec.Validations["paas"]["allowedQuotas"] = validation
				for _, test := range []v1alpha1.ConfigQuotaSettings{
					{DefQuota: invalidQuotas},
					{MinQuotas: invalidQuotas},
					{MaxQuotas: invalidQuotas},
				} {
					obj.Spec.Capabilities = v1alpha1.ConfigCapabilities{
						"quota": v1alpha1.ConfigCapability{
							QuotaSettings: test,
						},
					}
					warn, err := validator.ValidateCreate(ctx, obj)
					Expect(warn, err).Error().To(HaveOccurred())
				}
			})
		})
	})

	When("creating a new PaasConfig", func() {
		Context("having a capability defined with clusterwide=true", func() {
			It("should not check if Min > Def", func() {
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
		Context("having a capability defined with a custom_field", func() {
			It("should verify Validation field to be valid and default to meet validation", func() {
				tests := []struct {
					re        string
					valid     bool
					myDefault string
				}{
					{re: "^[0-9*$", valid: false},
					{re: "^[0-9]*$", valid: true},
					{re: "^[0-9]*$", myDefault: "1234", valid: true},
					{re: "^[0-9]*$", myDefault: "1a234", valid: false},
				}
				for _, test := range tests {
					fmt.Fprintf(GinkgoWriter, "DEBUG - Test: %v", test)
					obj.Spec.Capabilities = v1alpha1.ConfigCapabilities{
						"ValidatedCap": v1alpha1.ConfigCapability{
							AppSet: "custom-field-appset",
							QuotaSettings: v1alpha1.ConfigQuotaSettings{
								DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
									corev1.ResourceCPU: resourcev1.MustParse("5000m"),
								},
							},
							CustomFields: map[string]v1alpha1.ConfigCustomField{
								"with-validation": {
									Validation: test.re,
									Default:    test.myDefault,
								},
							},
						},
					}
					_, err := validator.ValidateCreate(ctx, obj)
					if test.valid {
						Expect(err).Error().NotTo(HaveOccurred())
					} else {
						Expect(err).Error().To(HaveOccurred())
					}
				}
			})
			It("should fail when any combination of required, default and template are set", func() {
				// Add cap for testing
				for _, test := range []v1alpha1.ConfigCustomField{
					{Default: "something", Required: true},
					{Template: "{{ .Paas.Metadata.Name }}", Required: true},
					{Template: "{{ .Paas.Metadata.Name }}", Default: "something"},
				} {
					fmt.Fprintf(GinkgoWriter, "DEBUG - Test: %v", test)
					obj.Spec.Capabilities = v1alpha1.ConfigCapabilities{
						"ValidatedCap": v1alpha1.ConfigCapability{
							AppSet: "custom-field-appset",
							QuotaSettings: v1alpha1.ConfigQuotaSettings{
								DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
									corev1.ResourceCPU: resourcev1.MustParse("5000m"),
								},
							},
							CustomFields: map[string]v1alpha1.ConfigCustomField{"to-test": test},
						},
					}
					_, err := validator.ValidateCreate(ctx, obj)
					Expect(err).Error().To(HaveOccurred())
				}
			})
			It("should verify Template field to be valid", func() {
				tests := []struct {
					template string
					valid    bool
				}{
					{template: "{{ .Paas.Name }}", valid: true},
					{template: "{{ .DoesNotExist }}", valid: true},
					{template: "{{ .MissingBrace }", valid: false},
					{template: "{{ range group in .Paas.Groups}}{{ .MissingEnd }}", valid: false},
				}
				for _, test := range tests {
					fmt.Fprintf(GinkgoWriter, "DEBUG - Test: %v", test)
					obj.Spec.Capabilities = v1alpha1.ConfigCapabilities{
						"CapWithTemplate": v1alpha1.ConfigCapability{
							AppSet: "custom-field-appset",
							QuotaSettings: v1alpha1.ConfigQuotaSettings{
								DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
									corev1.ResourceCPU: resourcev1.MustParse("5000m"),
								},
							},
							CustomFields: map[string]v1alpha1.ConfigCustomField{
								"with-template": {
									Template: test.template,
								},
							},
						},
					}
					_, err := validator.ValidateCreate(ctx, obj)
					if test.valid {
						Expect(err).Error().NotTo(HaveOccurred())
					} else {
						Expect(err).Error().To(HaveOccurred())
					}
				}
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
				DecryptKeysSecret: v1alpha1.NamespacedName{
					Name:      paasConfigPkSecret,
					Namespace: paasConfigSystem,
				},
				Validations: v1alpha1.PaasConfigValidations{
					"paas": {
						"groupNames": "[0-9a-z-]{1,63}",
					},
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
		Context("having a capability with invalid name", func() {
			It("Should validate capability names against validation in new config", func() {
				for _, test := range []struct {
					name       string
					validation string
					valid      bool
				}{
					{name: "valid-name", validation: "^[a-z-]+$", valid: true},
					{name: "invalid-name", validation: "^[a-z]+$", valid: false},
					{name: "", validation: "^.$", valid: false},
				} {
					obj.Spec.Capabilities = v1alpha1.ConfigCapabilities{
						test.name: v1alpha1.ConfigCapability{
							AppSet: "my-appset",
							QuotaSettings: v1alpha1.ConfigQuotaSettings{
								DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
									corev1.ResourceCPU: resourcev1.MustParse("5000m"),
								},
							},
						},
					}
					obj.Spec.Validations = v1alpha1.PaasConfigValidations{
						"paasConfig": {"capabilityName": test.validation},
					}
					warn, err := validator.ValidateCreate(ctx, obj)
					if test.valid {
						Expect(warn).To(BeEmpty())
						Expect(err).Error().NotTo(HaveOccurred())
					} else {
						Expect(warn).To(BeEmpty())
						Expect(err).Error().To(HaveOccurred())
					}
				}
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

var _ = Describe("Deleting a PaasConfig", Ordered, func() {
	var (
		obj       *v1alpha1.PaasConfig
		validator PaasConfigCustomValidator
	)

	BeforeEach(func() {
		obj = &v1alpha1.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "newPaasConfig"},
			Spec: v1alpha1.PaasConfigSpec{
				DecryptKeysSecret: v1alpha1.NamespacedName{
					Name:      paasConfigPkSecret,
					Namespace: paasConfigSystem,
				},
			},
		}
		validator = PaasConfigCustomValidator{client: k8sClient}
	})

	It("should not accept another resource type", func() {
		obj := &corev1.Secret{}
		warn, err := validator.ValidateDelete(ctx, obj)

		Expect(warn).To(BeEmpty())
		Expect(err).Error().To(MatchError("expected a PaasConfig object but got *v1.Secret"))
	})

	It("should not return warnings nor an error", func() {
		warn, err := validator.ValidateDelete(ctx, obj)

		Expect(warn).To(BeEmpty())
		Expect(err).Error().To(Not(HaveOccurred()))
	})
})

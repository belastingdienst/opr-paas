/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

// Excuse Ginkgo use from revive errors
//revive:disable:dot-imports

import (
	"encoding/base64"
	"errors"
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/belastingdienst/opr-paas/internal/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Paas Webhook", Ordered, func() {
	const paasName = "my-paas"
	var (
		obj       *v1alpha1.Paas
		oldObj    *v1alpha1.Paas
		validator PaasCustomValidator
		mycrypt   *crypt.Crypt
	)

	BeforeAll(func() {
		c, pkey, err := newGeneratedCrypt(paasName)
		Expect(err).NotTo(HaveOccurred())
		mycrypt = c

		createNamespace("paas-system")
		createPaasPrivateKeySecret("paas-system", "keys", pkey)
	})

	BeforeEach(func() {
		obj = &v1alpha1.Paas{}
		oldObj = &v1alpha1.Paas{}
		validator = PaasCustomValidator{k8sClient}
		conf := v1alpha1.PaasConfig{
			Spec: v1alpha1.PaasConfigSpec{
				DecryptKeysSecret: v1alpha1.NamespacedName{
					Name:      "keys",
					Namespace: "paas-system",
				},
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

			Expect(validator.ValidateCreate(ctx, obj)).Error().
				To(MatchError(ContainSubstring("capability not configured")))
		})

		It(
			"Should deny creation and return multiple field errors when multiple unconfigured capabilities are set",
			func() {
				obj = &v1alpha1.Paas{
					Spec: v1alpha1.PaasSpec{
						Capabilities: v1alpha1.PaasCapabilities{
							"foo": v1alpha1.PaasCapability{},
							"bar": v1alpha1.PaasCapability{},
						},
					},
				}

				_, err := validator.ValidateCreate(ctx, obj)
				Expect(err).Error().To(MatchError(ContainSubstring("Invalid value: \"foo\"")))
				Expect(err).Error().To(MatchError(ContainSubstring("Invalid value: \"bar\"")))
			},
		)

		It("Should deny creation when a secret is set that cannot be decrypted", func() {
			encrypted, err := mycrypt.Encrypt([]byte("some encrypted string"))
			Expect(err).NotTo(HaveOccurred())

			obj = &v1alpha1.Paas{
				ObjectMeta: metav1.ObjectMeta{Name: paasName},
				Spec: v1alpha1.PaasSpec{
					SshSecrets: map[string]string{
						"valid secret":   encrypted,
						"invalid secret": base64.StdEncoding.EncodeToString([]byte("foo bar baz")),
						"invalid base64": "foo bar baz",
					},
				},
			}

			_, err = validator.ValidateCreate(ctx, obj)
			var serr *apierrors.StatusError
			Expect(errors.As(err, &serr)).To(BeTrue())

			causes := serr.Status().Details.Causes
			Expect(causes).To(ContainElements(
				metav1.StatusCause{
					Type: metav1.CauseTypeFieldValueInvalid,
					// revive:disable-next-line
					Message: "Invalid value: \"invalid base64\": cannot be decrypted: illegal base64 data at input byte 8",
					Field:   "spec.sshSecrets",
				},
				metav1.StatusCause{
					Type: metav1.CauseTypeFieldValueInvalid,
					// revive:disable-next-line
					Message: "Invalid value: \"invalid secret\": cannot be decrypted: unable to decrypt data with any of the private keys",
					Field:   "spec.sshSecrets",
				},
			))
			Expect(causes).To(HaveLen(2))
		})

		It("Should deny creation when a capability custom field is not configured", func() {
			conf := config.GetConfig()
			conf.Capabilities["foo"] = v1alpha1.ConfigCapability{
				CustomFields: map[string]v1alpha1.ConfigCustomField{
					"bar": {},
				},
			}
			config.SetConfig(v1alpha1.PaasConfig{Spec: conf})

			obj = &v1alpha1.Paas{
				Spec: v1alpha1.PaasSpec{
					Capabilities: v1alpha1.PaasCapabilities{
						"foo": v1alpha1.PaasCapability{
							CustomFields: map[string]string{
								"bar": "baz",
								"baz": "qux",
							},
						},
					},
				},
			}
			_, err := validator.ValidateCreate(ctx, obj)

			var serr *apierrors.StatusError
			Expect(errors.As(err, &serr)).To(BeTrue())
			causes := serr.Status().Details.Causes
			Expect(causes).To(HaveLen(1))
			Expect(causes).To(ContainElements(
				metav1.StatusCause{
					Type: metav1.CauseTypeFieldValueInvalid,
					//revive:disable-next-line
					Message: "Invalid value: \"custom_fields\": custom field baz is not configured in capability config",
					Field:   "spec.capabilities[foo]",
				},
			))
		})

		It("Should deny creation when a capability is missing a required custom field", func() {
			conf := config.GetConfig()
			conf.Capabilities["foo"] = v1alpha1.ConfigCapability{
				CustomFields: map[string]v1alpha1.ConfigCustomField{
					"bar": {Required: true},
				},
			}
			config.SetConfig(v1alpha1.PaasConfig{Spec: conf})

			obj = &v1alpha1.Paas{
				Spec: v1alpha1.PaasSpec{
					Capabilities: v1alpha1.PaasCapabilities{
						"foo": v1alpha1.PaasCapability{},
					},
				},
			}
			_, err := validator.ValidateCreate(ctx, obj)

			var serr *apierrors.StatusError
			Expect(errors.As(err, &serr)).To(BeTrue())
			causes := serr.Status().Details.Causes
			Expect(causes).To(HaveLen(1))
			Expect(causes).To(ContainElements(
				metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "Invalid value: \"custom_fields\": value bar is required",
					Field:   "spec.capabilities[foo]",
				},
			))
		})

		It("Should deny creation when a custom field does not match validation regex", func() {
			conf := config.GetConfig()
			conf.Capabilities["foo"] = v1alpha1.ConfigCapability{
				CustomFields: map[string]v1alpha1.ConfigCustomField{
					"bar": {Validation: "^\\d+$"}, // Must be an integer
					"baz": {Validation: "^\\w+$"}, // Must not be whitespace
				},
			}
			config.SetConfig(v1alpha1.PaasConfig{Spec: conf})

			obj = &v1alpha1.Paas{
				Spec: v1alpha1.PaasSpec{
					Capabilities: v1alpha1.PaasCapabilities{
						"foo": v1alpha1.PaasCapability{
							CustomFields: map[string]string{
								"bar": "notinteger123",
								"baz": "word",
							},
						},
					},
				},
			}
			_, err := validator.ValidateCreate(ctx, obj)

			var serr *apierrors.StatusError
			Expect(errors.As(err, &serr)).To(BeTrue())
			causes := serr.Status().Details.Causes
			Expect(causes).To(HaveLen(1))
			Expect(causes).To(ContainElements(
				metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "Invalid value: \"custom_fields\": invalid value notinteger123 (does not match ^\\d+$)",
					Field:   "spec.capabilities[foo]",
				},
			))
		})

		It("Should allow groups with valid names", func() {
			var (
				validChars = "abcdefghijklmknopqrstuvwzyz-0123456789"
				validNames = []string{
					validChars,
					strings.Repeat(validChars, 3)[0:63],
				}
			)
			for _, validName := range validNames {
				validGroupNamePaas := &v1alpha1.Paas{
					Spec: v1alpha1.PaasSpec{
						Groups: map[string]v1alpha1.PaasGroup{
							validName: {
								Users: []string{"bar"},
							},
						},
					},
				}
				warnings, errors := validator.ValidateCreate(ctx, validGroupNamePaas)
				Expect(warnings).To(BeNil())
				Expect(errors).ToNot(HaveOccurred())
			}
		})

		// RFC 1035 Label Names
		//
		// Some resource types require their names to follow the DNS label standard as defined in RFC 1035.
		// This means the name must:
		//
		// contain at most 63 characters
		// contain only lowercase alphanumeric characters or '-'
		// start with an alphabetic character
		// end with an alphanumeric character
		It("Should deny groups with invalid names", func() {
			const (
				tooLongMsg      = "must be no more than 63 characters"
				invalidCharsMsg = "a DNS-1035 label must consist of lower case alphanumeric characters or '-'"
			)
			var (
				validChars   = "abcdefghijklmknopqrstuvwzyz-0123456789"
				invalidNames = map[string]string{
					"-" + validChars: invalidCharsMsg,
					validChars + "-": invalidCharsMsg,
					validChars[0:10] + "A" + validChars[11:20]: invalidCharsMsg,
					strings.Repeat(validChars, 3):              tooLongMsg,
				}
			)
			for invalidName, msg := range invalidNames {
				invalidGroupNamePaas := &v1alpha1.Paas{
					Spec: v1alpha1.PaasSpec{
						Groups: map[string]v1alpha1.PaasGroup{
							invalidName: {
								Users: []string{"bar"},
							},
						},
					},
				}
				warnings, errors := validator.ValidateCreate(ctx, invalidGroupNamePaas)
				Expect(warnings).To(BeEmpty())
				Expect(errors).To(MatchError(SatisfyAll(ContainSubstring(msg))))
			}

		})
		It("Should warn when a group contains both users and a query", func() {
			obj = &v1alpha1.Paas{
				Spec: v1alpha1.PaasSpec{
					Groups: map[string]v1alpha1.PaasGroup{
						"foo": {
							Users: []string{"bar"},
							Query: "baz",
						},
					},
				},
			}

			warnings, _ := validator.ValidateCreate(ctx, obj)
			Expect(
				warnings,
			).To(ContainElement("spec.groups[foo] contains both users and query, the users will be ignored"))
		})

		It("Should not warn when a group contains just users", func() {
			obj = &v1alpha1.Paas{
				Spec: v1alpha1.PaasSpec{
					Groups: map[string]v1alpha1.PaasGroup{
						"foo": {
							Users: []string{"bar"},
						},
					},
				},
			}

			warnings, _ := validator.ValidateCreate(ctx, obj)
			Expect(warnings).To(BeEmpty())
		})

		It("Should warn when quota limits are set higher than requests", func() {
			conf := config.GetConfig()
			conf.Capabilities["foo"] = v1alpha1.ConfigCapability{}
			config.SetConfig(v1alpha1.PaasConfig{Spec: conf})

			obj = &v1alpha1.Paas{
				Spec: v1alpha1.PaasSpec{
					Capabilities: v1alpha1.PaasCapabilities{
						"foo": v1alpha1.PaasCapability{
							Quota: quota.Quota{
								corev1.ResourceLimitsCPU:      resource.MustParse("2"),
								corev1.ResourceRequestsCPU:    resource.MustParse("2"),
								corev1.ResourceLimitsMemory:   resource.MustParse("256Mi"),
								corev1.ResourceRequestsMemory: resource.MustParse("1Gi"),
							},
						},
					},
					Quota: quota.Quota{
						corev1.ResourceLimitsCPU:   resource.MustParse("10"),
						corev1.ResourceRequestsCPU: resource.MustParse("11"),
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Not(HaveOccurred()))
			Expect(warnings).To(HaveLen(2))
			Expect(warnings).To(ContainElements(
				"spec.quota CPU resource request (11) higher than limit (10)",
				"spec.capabilities[foo].quota memory resource request (1Gi) higher than limit (256Mi)",
			))
		})

		It("Should warn when extra permissions are requested for a capability that are not configured", func() {
			conf := config.GetConfig()
			conf.Capabilities["foo"] = v1alpha1.ConfigCapability{
				ExtraPermissions: v1alpha1.ConfigCapPerm{
					"bar": []string{"baz"},
				},
			}
			conf.Capabilities["bar"] = v1alpha1.ConfigCapability{
				// No extra permissions
			}
			config.SetConfig(v1alpha1.PaasConfig{Spec: conf})

			obj = &v1alpha1.Paas{
				Spec: v1alpha1.PaasSpec{
					Capabilities: v1alpha1.PaasCapabilities{
						"foo": v1alpha1.PaasCapability{ExtraPermissions: true},
						"bar": v1alpha1.PaasCapability{ExtraPermissions: true},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Not(HaveOccurred()))
			Expect(warnings).To(HaveLen(1))
			//revive:disable-next-line
			Expect(warnings[0]).To(Equal("spec.capabilities[bar].extra_permissions capability does not have extra permissions configured"))
		})
	})

	Context("When updating a Paas under Validating Webhook", func() {
		It("Should deny creation when a capability is set that is not configured", func() {
			obj = &v1alpha1.Paas{Spec: v1alpha1.PaasSpec{
				Capabilities: v1alpha1.PaasCapabilities{"foo": v1alpha1.PaasCapability{}},
			}}

			Expect(validator.ValidateUpdate(ctx, nil, obj)).Error().
				To(MatchError(ContainSubstring("capability not configured")))
		})

		It(
			"Should generate a warning when updating a Paas with a group that contains both users and a queries",
			func() {
				obj = &v1alpha1.Paas{
					Spec: v1alpha1.PaasSpec{
						Groups: map[string]v1alpha1.PaasGroup{
							"foo": {
								Users: []string{"bar"},
								Query: "baz",
							},
						},
					},
				}

				warnings, _ := validator.ValidateUpdate(ctx, nil, obj)
				Expect(
					warnings,
				).To(ContainElement("spec.groups[foo] contains both users and query, the users will be ignored"))
			},
		)

		It("Should not warn when updating a Paas with a group that contains just users", func() {
			obj = &v1alpha1.Paas{
				Spec: v1alpha1.PaasSpec{
					Groups: map[string]v1alpha1.PaasGroup{
						"foo": {
							Users: []string{"bar"},
						},
					},
				},
			}

			warnings, _ := validator.ValidateUpdate(ctx, nil, obj)
			Expect(warnings).To(BeEmpty())
		})
	})
})

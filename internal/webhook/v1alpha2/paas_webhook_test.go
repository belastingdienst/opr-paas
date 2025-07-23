/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

// Excuse Ginkgo use from revive errors
// revive:disable:dot-imports

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/config"
	"github.com/belastingdienst/opr-paas/v3/pkg/quota"
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
		obj       *v1alpha2.Paas
		oldObj    *v1alpha2.Paas
		validator PaasCustomValidator
		mycrypt   *crypt.Crypt
		conf      v1alpha2.PaasConfig
	)

	BeforeAll(func() {
		c, pkey, err := newGeneratedCrypt(paasName)
		Expect(err).NotTo(HaveOccurred())
		mycrypt = c

		createNamespace(k8sClient, "paas-system")
		createPaasPrivateKeySecret(k8sClient, "paas-system", "keys", pkey)
	})

	BeforeEach(func() {
		obj = &v1alpha2.Paas{}
		oldObj = &v1alpha2.Paas{}
		validator = PaasCustomValidator{k8sClient}
		conf = v1alpha2.PaasConfig{
			Spec: v1alpha2.PaasConfigSpec{
				DecryptKeysSecret: v1alpha2.NamespacedName{
					Name:      "keys",
					Namespace: "paas-system",
				},
				Capabilities: v1alpha2.ConfigCapabilities{
					"cap5": v1alpha2.ConfigCapability{
						AppSet: "someAppset",
						QuotaSettings: v1alpha2.ConfigQuotaSettings{
							DefQuota: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceLimitsCPU: resource.MustParse("5"),
							},
						},
					},
				},
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
		It("Should allow creation of a valid Paas", func() {
			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Capabilities: v1alpha2.PaasCapabilities{"cap5": v1alpha2.PaasCapability{}},
				},
			}

			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().NotTo(HaveOccurred())
			Expect(err).Error().NotTo(HaveOccurred())
		})
		It("Should validate paas name", func() {
			const paasNameValidation = "^([a-z0-9]{3})-([a-z0-9]{3})$"
			conf.Spec.Validations = v1alpha2.PaasConfigValidations{"paas": {"name": paasNameValidation}}

			config.SetConfig(conf)
			obj = &v1alpha2.Paas{
				ObjectMeta: metav1.ObjectMeta{
					Name: "this-is-invalid",
				},
				Spec: v1alpha2.PaasSpec{
					Capabilities: v1alpha2.PaasCapabilities{"foo": v1alpha2.PaasCapability{}},
				},
			}

			Expect(validator.ValidateCreate(ctx, obj)).Error().
				To(MatchError(ContainSubstring("capability not configured")))
		})
		It("Should validate the requestor field", func() {
			for _, test := range []struct {
				requestor  string
				validation string
				valid      bool
			}{
				{requestor: "valid-requestor", validation: "^[a-z-]+$", valid: true},
				{requestor: "invalid-requestor", validation: "^[a-z]+$", valid: false},
				{requestor: "", validation: "^.$", valid: false},
			} {
				fmt.Fprintf(GinkgoWriter, "DEBUG - Test: %v", test)
				conf.Spec.Validations = v1alpha2.PaasConfigValidations{"paas": {"requestor": test.validation}}
				config.SetConfig(conf)
				obj.Spec.Requestor = test.requestor
				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(warn).To(BeNil())
				if test.valid {
					Expect(warn, err).Error().NotTo(HaveOccurred())
				} else {
					Expect(warn, err).Error().To(HaveOccurred())
				}
			}
		})
		It("Should validate namespace names", func() {
			for _, test := range []struct {
				name       string
				validation string
				valid      bool
			}{
				{name: "valid-name", validation: "^[a-z-]+$", valid: true},
				{name: "invalid-name", validation: "^[a-z]+$", valid: false},
				{name: "", validation: "^.$", valid: false},
			} {
				fmt.Fprintf(GinkgoWriter, "DEBUG - Test: %v", test)
				conf.Spec.Validations = v1alpha2.PaasConfigValidations{"paas": {"namespaceName": test.validation}}
				config.SetConfig(conf)
				obj.Spec.Namespaces = v1alpha2.PaasNamespaces{
					test.name: v1alpha2.PaasNamespace{},
				}
				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(warn).To(BeNil())
				if test.valid {
					Expect(warn, err).Error().NotTo(HaveOccurred())
				} else {
					Expect(warn, err).Error().To(HaveOccurred())
				}
			}
		})
		It("Should deny creation when a capability is set that is not configured", func() {
			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Capabilities: v1alpha2.PaasCapabilities{"foo": v1alpha2.PaasCapability{}},
				},
			}

			Expect(validator.ValidateCreate(ctx, obj)).Error().
				To(MatchError(ContainSubstring("capability not configured")))
		})

		It(
			"Should deny creation and return multiple field errors when multiple unconfigured capabilities are set",
			func() {
				obj = &v1alpha2.Paas{
					Spec: v1alpha2.PaasSpec{
						Capabilities: v1alpha2.PaasCapabilities{
							"foo": v1alpha2.PaasCapability{},
							"bar": v1alpha2.PaasCapability{},
						},
					},
				}

				_, err := validator.ValidateCreate(ctx, obj)
				Expect(err).Error().To(MatchError(ContainSubstring("Invalid value: \"foo\"")))
				Expect(err).Error().To(MatchError(ContainSubstring("Invalid value: \"bar\"")))
			},
		)

		It("Should deny creation when a capability secret is set that cannot be decrypted", func() {
			encrypted, err := mycrypt.Encrypt([]byte("some encrypted string"))
			Expect(err).NotTo(HaveOccurred())

			conf := config.GetConfig().Spec
			conf.Capabilities["foo"] = v1alpha2.ConfigCapability{}
			config.SetConfig(v1alpha2.PaasConfig{Spec: conf})

			obj = &v1alpha2.Paas{
				ObjectMeta: metav1.ObjectMeta{Name: paasName},
				Spec: v1alpha2.PaasSpec{
					Secrets: map[string]string{
						"valid secret":   encrypted,
						"invalid secret": base64.StdEncoding.EncodeToString([]byte("foo bar baz")),
						"invalid base64": "foo bar baz",
					},
					Capabilities: map[string]v1alpha2.PaasCapability{
						"foo": {
							Secrets: map[string]string{
								"valid secret":   encrypted,
								"invalid secret": base64.StdEncoding.EncodeToString([]byte("foo bar baz")),
								"invalid base64": "foo bar baz",
							},
						},
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
					Message: "Invalid value: \"foo bar baz\": cannot be decrypted: " +
						"illegal base64 data at input byte 8",
					Field: "spec.capabilities[foo].secrets[invalid base64]",
				},
				metav1.StatusCause{
					Type: metav1.CauseTypeFieldValueInvalid,
					Message: "Invalid value: \"Zm9vIGJhciBiYXo=\": cannot be decrypted: " +
						"unable to decrypt data with any of the private keys",
					Field: "spec.capabilities[foo].secrets[invalid secret]",
				},
				metav1.StatusCause{
					Type: metav1.CauseTypeFieldValueInvalid,
					Message: "Invalid value: \"foo bar baz\": cannot be decrypted: " +
						"illegal base64 data at input byte 8",
					Field: "spec.secrets[invalid base64]",
				},
				metav1.StatusCause{
					Type: metav1.CauseTypeFieldValueInvalid,
					Message: "Invalid value: \"Zm9vIGJhciBiYXo=\": cannot be decrypted: " +
						"unable to decrypt data with any of the private keys",
					Field: "spec.secrets[invalid secret]",
				},
			))
			Expect(causes).To(HaveLen(4))
		})

		It("Should deny creation when a capability custom field is not configured", func() {
			conf := config.GetConfig().Spec
			conf.Capabilities["foo"] = v1alpha2.ConfigCapability{
				CustomFields: map[string]v1alpha2.ConfigCustomField{
					"bar": {},
				},
			}
			config.SetConfig(v1alpha2.PaasConfig{Spec: conf})

			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Capabilities: v1alpha2.PaasCapabilities{
						"foo": v1alpha2.PaasCapability{
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
					Message: "Invalid value: \"custom_fields\": " +
						"custom field baz is not configured in capability config",
					Field: "spec.capabilities[foo]",
				},
			))
		})

		It("Should deny creation when a capability is missing a required custom field", func() {
			conf := config.GetConfig().Spec
			conf.Capabilities["foo"] = v1alpha2.ConfigCapability{
				CustomFields: map[string]v1alpha2.ConfigCustomField{
					"bar": {Required: true},
				},
			}
			config.SetConfig(v1alpha2.PaasConfig{Spec: conf})

			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Capabilities: v1alpha2.PaasCapabilities{
						"foo": v1alpha2.PaasCapability{},
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
			// conf := config.GetVersionedConfig[v1alpha2.PaasConfig]().Spec
			conf := config.GetConfig().Spec
			conf.Capabilities["foo"] = v1alpha2.ConfigCapability{
				CustomFields: map[string]v1alpha2.ConfigCustomField{
					"bar": {Validation: "^\\d+$"}, // Must be an integer
					"baz": {Validation: "^\\w+$"}, // Must not be whitespace
				},
			}
			config.SetConfig(v1alpha2.PaasConfig{Spec: conf})

			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Capabilities: v1alpha2.PaasCapabilities{
						"foo": v1alpha2.PaasCapability{
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

		It("Should deny creation when a namespace group does not match any Paas group", func() {
			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Groups: v1alpha2.PaasGroups{
						"group1": {},
						"group2": {},
					},
					Namespaces: v1alpha2.PaasNamespaces{
						"foo": {
							Groups: []string{"group2", "group3"},
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
					Message: "Invalid value: \"group3\": does not exist in paas groups (group1, group2)",
					Field:   "spec.namespaces[foo].groups",
				},
			))
		})

		It("Should validate group names", func() {
			conf.Spec.Validations = v1alpha2.PaasConfigValidations{"paas": {"groupName": "^[a-z0-9-]{1,63}$"}}
			config.SetConfig(conf)
			validChars := "abcdefghijklmknopqrstuvwzyz-0123456789"
			for _, test := range []struct {
				name  string
				valid bool
			}{
				{name: validChars, valid: true},
				{name: strings.Repeat(validChars, 3)[0:63], valid: true},
				{name: "_" + validChars, valid: false},
				{name: "A" + validChars, valid: false},
				{name: strings.Repeat(validChars, 3)[0:64], valid: false},
			} {
				validGroupNamePaas := &v1alpha2.Paas{
					Spec: v1alpha2.PaasSpec{
						Groups: map[string]v1alpha2.PaasGroup{
							test.name: {
								Users: []string{"bar"},
							},
						},
					},
				}
				warnings, errors := validator.ValidateCreate(ctx, validGroupNamePaas)
				if test.valid {
					Expect(warnings).To(BeNil())
					Expect(errors).ToNot(HaveOccurred())
				} else {
					Expect(warnings, errors).Error().To(HaveOccurred())
					Expect(errors).To(MatchError(SatisfyAll(
						ContainSubstring("group name does not match configured validation regex"))))
				}
			}
		})

		It("Should warn when a group contains both users and a query", func() {
			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Groups: map[string]v1alpha2.PaasGroup{
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
			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Groups: map[string]v1alpha2.PaasGroup{
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
			conf := config.GetConfig().Spec
			conf.Capabilities["foo"] = v1alpha2.ConfigCapability{}
			config.SetConfig(v1alpha2.PaasConfig{Spec: conf})

			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Capabilities: v1alpha2.PaasCapabilities{
						"foo": v1alpha2.PaasCapability{
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
			conf := config.GetConfig().Spec
			conf.Capabilities["foo"] = v1alpha2.ConfigCapability{
				ExtraPermissions: v1alpha2.ConfigCapPerm{
					"bar": []string{"baz"},
				},
			}
			conf.Capabilities["bar"] = v1alpha2.ConfigCapability{
				// No extra permissions
			}
			config.SetConfig(v1alpha2.PaasConfig{Spec: conf})

			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Capabilities: v1alpha2.PaasCapabilities{
						"foo": v1alpha2.PaasCapability{ExtraPermissions: true},
						"bar": v1alpha2.PaasCapability{ExtraPermissions: true},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Not(HaveOccurred()))
			Expect(warnings).To(HaveLen(1))
			Expect(warnings[0]).To(
				Equal("spec.capabilities[bar].extra_permissions capability " +
					"does not have extra permissions configured"),
			)
		})
		It("Should handle user feature flag properly", func() {
			for setting, expects := range map[string]struct {
				warn string
				err  string
			}{
				"":      {},
				"allow": {},
				"warn":  {warn: "group spec.groups[foo].users has users which is discouraged"},
				"block": {err: "groups with users is a disabled feature"},
			} {
				fmt.Fprintf(GinkgoWriter, "DEBUG - Test: %s: %s", setting, expects)
				conf.Spec.FeatureFlags.GroupUserManagement = setting
				config.SetConfig(conf)
				obj = &v1alpha2.Paas{
					Spec: v1alpha2.PaasSpec{
						Groups: map[string]v1alpha2.PaasGroup{
							"foo": {
								Users: []string{"bar"},
							},
						},
					},
				}
				warnings, err := validator.ValidateCreate(ctx, obj)
				if expects.warn == "" {
					Expect(warnings).To(BeEmpty())
				} else {
					Expect(warnings).To(ContainElement(expects.warn))
				}
				if expects.err == "" {
					Expect(err).NotTo(HaveOccurred())
				} else {
					Expect(err).To(MatchError(SatisfyAll(ContainSubstring(expects.err))))
				}
			}
		})
	})

	Context("When updating a Paas under Validating Webhook", func() {
		It("Should deny creation when a capability is set that is not configured", func() {
			obj = &v1alpha2.Paas{Spec: v1alpha2.PaasSpec{
				Capabilities: v1alpha2.PaasCapabilities{"foo": v1alpha2.PaasCapability{}},
			}}

			Expect(validator.ValidateUpdate(ctx, nil, obj)).Error().
				To(MatchError(ContainSubstring("capability not configured")))
		})

		It(
			"Should generate a warning when updating a Paas with a group that contains both users and a queries",
			func() {
				obj = &v1alpha2.Paas{
					Spec: v1alpha2.PaasSpec{
						Groups: map[string]v1alpha2.PaasGroup{
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
			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Groups: map[string]v1alpha2.PaasGroup{
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

func newGeneratedCrypt(context string) (myCrypt *crypt.Crypt, privateKey []byte, err error) {
	tmpFileError := "failed to get new tmp private key file: %w"
	privateKeyFile, err := os.CreateTemp("", "private")
	if err != nil {
		return nil, nil, fmt.Errorf(tmpFileError, err)
	}
	publicKeyFile, err := os.CreateTemp("", "public")
	if err != nil {
		return nil, nil, fmt.Errorf(tmpFileError, err)
	}
	myCrypt, err = crypt.NewGeneratedCrypt(privateKeyFile.Name(), publicKeyFile.Name(), context)
	if err != nil {
		return nil, nil, fmt.Errorf(tmpFileError, err)
	}
	privateKey, err = os.ReadFile(privateKeyFile.Name())
	if err != nil {
		return nil, nil, errors.New("failed to read private key from file")
	}

	return myCrypt, privateKey, nil
}

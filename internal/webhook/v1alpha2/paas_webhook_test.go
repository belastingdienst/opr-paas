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
	"github.com/belastingdienst/opr-paas/v3/pkg/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	cl "sigs.k8s.io/controller-runtime/pkg/client"
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
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
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

		err := k8sClient.Create(ctx, &conf)
		Expect(err).NotTo(HaveOccurred())

		latest := &v1alpha2.PaasConfig{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latest)
		Expect(err).NotTo(HaveOccurred())

		meta.SetStatusCondition(&latest.Status.Conditions, metav1.Condition{
			Type:   v1alpha2.TypeActivePaasConfig,
			Status: metav1.ConditionTrue, Reason: "Reconciling", ObservedGeneration: latest.Generation,
			Message: "This config is the active config!",
		})

		err = k8sClient.Status().Update(ctx, latest)
		Expect(err).NotTo(HaveOccurred())

		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		err := k8sClient.Delete(ctx, &conf)
		Expect(err).NotTo(HaveOccurred())
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
		It("should allow creating a paas with namespaces", func() {
			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Capabilities: v1alpha2.PaasCapabilities{"cap5": v1alpha2.PaasCapability{}},
					Namespaces:   v1alpha2.PaasNamespaces{"myns": v1alpha2.PaasNamespace{}},
					Quota:        quota.Quota{"cpu.limits": resource.MustParse("10")},
				},
			}

			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().NotTo(HaveOccurred())
			Expect(err).Error().NotTo(HaveOccurred())
		})
		It("Should validate paas name", func() {
			const paasNameValidation = "^([a-z0-9]{3})-([a-z0-9]{3})$"

			// Update PaasConfig
			latestConf := &v1alpha2.PaasConfig{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
			Expect(err).To(Not(HaveOccurred()))
			latestConf.Spec.Validations = v1alpha2.PaasConfigValidations{"paas": {"name": paasNameValidation}}
			err = k8sClient.Update(ctx, latestConf)
			Expect(err).To(Not(HaveOccurred()))

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
				// Update PaasConfig
				latestConf := &v1alpha2.PaasConfig{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
				Expect(err).To(Not(HaveOccurred()))
				patch := cl.MergeFrom(latestConf.DeepCopy())
				latestConf.Spec.Validations = v1alpha2.PaasConfigValidations{"paas": {"requestor": test.validation}}
				err = k8sClient.Patch(ctx, latestConf, patch)
				Expect(err).To(Not(HaveOccurred()))

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

				// Update PaasConfig
				latestConf := &v1alpha2.PaasConfig{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
				Expect(err).To(Not(HaveOccurred()))
				latestConf.Spec.Validations = v1alpha2.PaasConfigValidations{"paas": {"namespaceName": test.validation}}
				err = k8sClient.Update(ctx, latestConf)
				Expect(err).To(Not(HaveOccurred()))

				obj.Spec.Namespaces = v1alpha2.PaasNamespaces{
					test.name: v1alpha2.PaasNamespace{},
				}
				obj.Spec.Quota = quota.Quota{
					"limits.cpu": resource.MustParse("1"),
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

			// Update PaasConfig
			latestConf := &v1alpha2.PaasConfig{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
			Expect(err).To(Not(HaveOccurred()))
			latestConf.Spec.Capabilities["foo"] = v1alpha2.ConfigCapability{
				QuotaSettings: v1alpha2.ConfigQuotaSettings{
					DefQuota: map[corev1.ResourceName]resource.Quantity{"foo": resource.MustParse("1")},
				},
			}
			err = k8sClient.Update(ctx, latestConf)
			Expect(err).To(Not(HaveOccurred()))

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
			// Update PaasConfig
			latestConf := &v1alpha2.PaasConfig{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
			Expect(err).To(Not(HaveOccurred()))

			latestConf.Spec.Capabilities["foo"] = v1alpha2.ConfigCapability{
				QuotaSettings: v1alpha2.ConfigQuotaSettings{
					DefQuota: map[corev1.ResourceName]resource.Quantity{"foo": resource.MustParse("1")},
				},
				CustomFields: map[string]v1alpha2.ConfigCustomField{
					"bar": {},
				},
			}
			err = k8sClient.Update(ctx, latestConf)
			Expect(err).To(Not(HaveOccurred()))

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
			_, err = validator.ValidateCreate(ctx, obj)

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
			// Update PaasConfig
			latestConf := &v1alpha2.PaasConfig{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
			Expect(err).To(Not(HaveOccurred()))
			latestConf.Spec.Capabilities["foo"] = v1alpha2.ConfigCapability{
				QuotaSettings: v1alpha2.ConfigQuotaSettings{
					DefQuota: map[corev1.ResourceName]resource.Quantity{"foo": resource.MustParse("1")},
				},
				CustomFields: map[string]v1alpha2.ConfigCustomField{
					"bar": {Required: true},
				},
			}
			err = k8sClient.Update(ctx, latestConf)
			Expect(err).To(Not(HaveOccurred()))

			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Capabilities: v1alpha2.PaasCapabilities{
						"foo": v1alpha2.PaasCapability{},
					},
				},
			}
			_, err = validator.ValidateCreate(ctx, obj)

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
			// Update PaasConfig
			latestConf := &v1alpha2.PaasConfig{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
			Expect(err).To(Not(HaveOccurred()))
			latestConf.Spec.Capabilities["foo"] = v1alpha2.ConfigCapability{
				QuotaSettings: v1alpha2.ConfigQuotaSettings{
					DefQuota: map[corev1.ResourceName]resource.Quantity{"foo": resource.MustParse("1")},
				},
				CustomFields: map[string]v1alpha2.ConfigCustomField{
					"bar": {Validation: "^\\d+$"}, // Must be an integer
					"baz": {Validation: "^\\w+$"}, // Must not be whitespace
				},
			}
			err = k8sClient.Update(ctx, latestConf)
			Expect(err).To(Not(HaveOccurred()))

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
			_, err = validator.ValidateCreate(ctx, obj)

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
					Quota: quota.Quota{
						corev1.ResourceLimitsCPU: resource.MustParse("1"),
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
			// Update PaasConfig
			latestConf := &v1alpha2.PaasConfig{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
			Expect(err).To(Not(HaveOccurred()))
			latestConf.Spec.Validations = v1alpha2.PaasConfigValidations{"paas": {"groupName": "^[a-z0-9-]{1,63}$"}}
			err = k8sClient.Update(ctx, latestConf)
			Expect(err).To(Not(HaveOccurred()))

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
			// Update PaasConfig
			latestConf := &v1alpha2.PaasConfig{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
			Expect(err).To(Not(HaveOccurred()))
			latestConf.Spec.Capabilities["foo"] = v1alpha2.ConfigCapability{
				QuotaSettings: v1alpha2.ConfigQuotaSettings{
					DefQuota: map[corev1.ResourceName]resource.Quantity{"foo": resource.MustParse("1")},
				},
			}
			err = k8sClient.Update(ctx, latestConf)
			Expect(err).To(Not(HaveOccurred()))

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
			// Update PaasConfig
			latestConf := &v1alpha2.PaasConfig{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
			Expect(err).To(Not(HaveOccurred()))
			latestConf.Spec.Capabilities["foo"] = v1alpha2.ConfigCapability{
				QuotaSettings: v1alpha2.ConfigQuotaSettings{
					DefQuota: map[corev1.ResourceName]resource.Quantity{"foo": resource.MustParse("1")},
				},
				ExtraPermissions: v1alpha2.ConfigCapPerm{
					"bar": []string{"baz"},
				},
			}
			latestConf.Spec.Capabilities["bar"] = v1alpha2.ConfigCapability{
				QuotaSettings: v1alpha2.ConfigQuotaSettings{
					DefQuota: map[corev1.ResourceName]resource.Quantity{"bar": resource.MustParse("1")},
				},
			}
			err = k8sClient.Update(ctx, latestConf)
			Expect(err).To(Not(HaveOccurred()))

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
				// Update PaasConfig
				latestConf := &v1alpha2.PaasConfig{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
				Expect(err).To(Not(HaveOccurred()))
				latestConf.Spec.FeatureFlags.GroupUserManagement = setting
				err = k8sClient.Update(ctx, latestConf)
				Expect(err).To(Not(HaveOccurred()))
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
				validQuotas = quota.Quota{
					"limits.memory":    resource.MustParse("100M"),
					"requests.cpu":     resource.MustParse("1.1"),
					"requests.memory":  resource.MustParse("100M"),
					"requests.storage": resource.MustParse("10G"),
					"thin.storageclass.storage.k8s.io/persistentvolumeclaims": resource.MustParse("10G"),
				}
				validation       = fmt.Sprintf("(%s)", strings.Join(validResourceKeys, "|"))
				validationConfig = v1alpha2.PaasConfigValidations{
					"paas": v1alpha2.PaasConfigTypeValidations{"allowedQuotas": validation},
				}
				invalidQuotas = map[corev1.ResourceName]resource.Quantity{
					"limits.cpu": resource.MustParse("1.1"),
				}
			)
			It("should allow cap quota names that meet re", func() {
				// Update PaasConfig
				latestConf := &v1alpha2.PaasConfig{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
				Expect(err).To(Not(HaveOccurred()))
				latestConf.Spec.Validations = validationConfig
				err = k8sClient.Update(ctx, latestConf)
				Expect(err).To(Not(HaveOccurred()))
				obj = &v1alpha2.Paas{
					Spec: v1alpha2.PaasSpec{
						Capabilities: v1alpha2.PaasCapabilities{
							"cap5": v1alpha2.PaasCapability{
								Quota: validQuotas,
							},
						},
					},
				}
				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(warn, err).Error().NotTo(HaveOccurred())
				Expect(err).Error().NotTo(HaveOccurred())
			})
			It("should deny cap quota names that do not meet re", func() {
				// Update PaasConfig
				latestConf := &v1alpha2.PaasConfig{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
				Expect(err).To(Not(HaveOccurred()))
				latestConf.Spec.Validations = validationConfig
				err = k8sClient.Update(ctx, latestConf)
				Expect(err).To(Not(HaveOccurred()))
				obj = &v1alpha2.Paas{
					Spec: v1alpha2.PaasSpec{
						Capabilities: v1alpha2.PaasCapabilities{
							"cap5": v1alpha2.PaasCapability{
								Quota: invalidQuotas,
							},
						},
					},
				}
				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(warn, err).Error().To(HaveOccurred())
			})
			It("should allow paas quota names that meet re", func() {
				// Update PaasConfig
				latestConf := &v1alpha2.PaasConfig{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
				Expect(err).To(Not(HaveOccurred()))
				latestConf.Spec.Validations = validationConfig
				err = k8sClient.Update(ctx, latestConf)
				Expect(err).To(Not(HaveOccurred()))
				obj = &v1alpha2.Paas{Spec: v1alpha2.PaasSpec{Quota: validQuotas}}
				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(warn, err).Error().NotTo(HaveOccurred())
				Expect(err).Error().NotTo(HaveOccurred())
			})
			It("should deny cap quota names that do not meet re", func() {
				// Update PaasConfig
				latestConf := &v1alpha2.PaasConfig{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latestConf)
				Expect(err).To(Not(HaveOccurred()))
				latestConf.Spec.Validations = validationConfig
				err = k8sClient.Update(ctx, latestConf)
				Expect(err).To(Not(HaveOccurred()))
				obj = &v1alpha2.Paas{Spec: v1alpha2.PaasSpec{Quota: invalidQuotas}}
				warn, err := validator.ValidateCreate(ctx, obj)
				Expect(warn, err).Error().To(HaveOccurred())
			})
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
		It("Should deny creation when a namespace is defined but no quota", func() {
			obj = &v1alpha2.Paas{
				Spec: v1alpha2.PaasSpec{
					Namespaces: v1alpha2.PaasNamespaces{
						"bobber": {},
					},
				},
			}
			_, err := validator.ValidateCreate(ctx, obj)

			// We want to assert the actual message of the cause, which is packed in the StatusError object type.
			// Therefore we type assert, unpack and assert that the Cause is exactly as expected
			var serr *apierrors.StatusError
			Expect(errors.As(err, &serr)).To(BeTrue())
			causes := serr.Status().Details.Causes
			Expect(causes).To(HaveLen(1))
			Expect(causes).To(ContainElements(
				metav1.StatusCause{
					Type: "FieldValueInvalid",
					Message: "Invalid value: \"1\": quota can not be empty when paas has namespaces" +
						" (number of namespaces: 1)",
					Field: "spec.namespaces",
				},
			))
		})
		It("Should deny modification when a capability with paasNS has been modified to have no quota", func() {
			obj = &v1alpha2.Paas{
				ObjectMeta: metav1.ObjectMeta{Name: paasName},
				Spec: v1alpha2.PaasSpec{
					Capabilities: v1alpha2.PaasCapabilities{"cap5": v1alpha2.PaasCapability{}},
					Quota:        quota.Quota{"limits.cpu": resource.MustParse("10")},
				},
			}
			Expect(k8sClient.Create(ctx, obj)).Error().NotTo(HaveOccurred())

			nsObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "my-paas-cap5"}}
			Expect(k8sClient.Create(ctx, nsObj)).Error().NotTo(HaveOccurred())

			paasNsObj := &v1alpha2.PaasNS{ObjectMeta: metav1.ObjectMeta{Name: "my-paasns", Namespace: "my-paas-cap5"}}
			Expect(k8sClient.Create(ctx, paasNsObj)).Error().NotTo(HaveOccurred())

			newObj := obj.DeepCopy()
			newObj.Spec.Quota = nil
			_, err := validator.ValidateUpdate(ctx, obj, newObj)

			// We want to assert the actual message of the cause, which is packed in the StatusError object type.
			// Therefore we type assert, unpack and assert that the Cause is exactly as expected
			var serr *apierrors.StatusError
			Expect(errors.As(err, &serr)).To(BeTrue())
			causes := serr.Status().Details.Causes
			Expect(causes).To(HaveLen(1))
			Expect(causes).To(ContainElements(
				metav1.StatusCause{
					Type:    "FieldValueInvalid",
					Message: `Invalid value: "1": quota can not be empty when paas capability namespace has paasNs`,
					Field:   "spec.capabilities[cap5]",
				},
			))
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

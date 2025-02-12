/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"encoding/base64"
	"errors"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		It("Should deny creation and return multiple field errors when multiple unconfigured capabilities are set", func() {
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
		})

		It("Should deny creation when a secret is set that cannot be decrypted", func() {
			const paasName = "my-paas"
			encrypted, err := rsa.EncryptOAEP(sha512.New(), rand.Reader, pubkey, []byte("some encrypted string"), []byte(paasName))
			Expect(err).NotTo(HaveOccurred())

			obj = &v1alpha1.Paas{
				ObjectMeta: metav1.ObjectMeta{Name: paasName},
				Spec: v1alpha1.PaasSpec{
					SshSecrets: map[string]string{
						"valid secret":   base64.StdEncoding.EncodeToString(encrypted),
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
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "Invalid value: \"invalid base64\": cannot be decrypted: illegal base64 data at input byte 8",
					Field:   "spec.sshSecrets",
				},
				metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "Invalid value: \"invalid secret\": cannot be decrypted: unable to decrypt data with any of the private keys",
					Field:   "spec.sshSecrets",
				},
			))
			Expect(causes).To(HaveLen(2))
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
			Expect(validator.ValidateUpdate(ctx, oldObj, obj)).Error().
				To(MatchError(ContainSubstring("capability not configured")))
		})

		// It("Should validate updates correctly", func() {
		//     By("simulating a valid update scenario")
		//     oldObj.SomeRequiredField = "updated_value"
		//     obj.SomeRequiredField = "updated_value"
		//     Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil())
		// })
	})
})

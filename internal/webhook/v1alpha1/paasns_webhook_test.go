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
	"time"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("PaasNS Webhook", Ordered, func() {
	const (
		paasSystem        = "paasnssystem"
		paasPkSecret      = "paasns-pk-secret"
		paasName          = "mypaas"
		otherPaasName     = "myotherpaas"
		nsWithoutOwnerRef = paasName + "-without-owner-ref"
		groupName1        = "mygroup1"
		groupName2        = "mygroup2"
		otherGroup        = "myothergroup"
		paasNsName        = "mypaasns"
	)
	var (
		privateKey      []byte
		paas            *v1alpha1.Paas
		obj             *v1alpha1.PaasNS
		oldObj          *v1alpha1.PaasNS
		mycrypt         *crypt.Crypt
		paasSecret      string
		validSecret1    string
		validSecret1Key = "validSecret1"
		validSecret2    string
		validator       PaasNSCustomValidator
		conf            v1alpha1.PaasConfig
	)

	BeforeAll(func() {
		var err error

		conf = v1alpha1.PaasConfig{
			Spec: v1alpha1.PaasConfigSpec{
				DecryptKeysSecret: v1alpha1.NamespacedName{
					Name:      paasPkSecret,
					Namespace: paasSystem,
				},
			},
		}
		config.SetConfigV1(conf)
		createNamespace(paasSystem)

		mycrypt, privateKey, err = newGeneratedCrypt(paasName)
		if err != nil {
			Fail(err.Error())
		}

		createPaasPrivateKeySecret(paasSystem, paasPkSecret, privateKey)
		paasSecret, err = mycrypt.Encrypt([]byte("paaSecret"))
		Expect(err).NotTo(HaveOccurred())
		validSecret1, err = mycrypt.Encrypt([]byte("validSecretCreated"))
		Expect(err).NotTo(HaveOccurred())
		validSecret2, err = mycrypt.Encrypt([]byte("validSecretUpdated"))
		Expect(err).NotTo(HaveOccurred())
		paas = &v1alpha1.Paas{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Paas",
				APIVersion: "v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
			},
			Spec: v1alpha1.PaasSpec{
				Requestor: "paasns-requestor",
				Groups: v1alpha1.PaasGroups{
					groupName1: v1alpha1.PaasGroup{},
					groupName2: v1alpha1.PaasGroup{},
				},
				SSHSecrets: map[string]string{
					paasSecret: paasSecret,
				},
				Quota: quota.Quota{
					"cpu": resource.MustParse("1"),
				},
			},
		}
		err = k8sClient.Create(ctx, paas)
		Expect(err).NotTo(HaveOccurred())
		retrievedPaas := &v1alpha1.Paas{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: paasName}, retrievedPaas)
		Expect(err).NotTo(HaveOccurred())
		Expect(retrievedPaas.Name).To(Equal(paasName))
		createPaasNamespace(*retrievedPaas, paasName)
		// This is a namespace with Owner ref to mypaas and prefix set to myotherpaas-
		createPaasNamespace(*retrievedPaas, otherPaasName)
		// This is a namespace without Owner ref
		createNamespace(nsWithoutOwnerRef)
	})

	BeforeEach(func() {
		obj = &v1alpha1.PaasNS{
			ObjectMeta: metav1.ObjectMeta{
				Name:      paasNsName,
				Namespace: paasName,
			},
			Spec: v1alpha1.PaasNSSpec{
				Paas: paasName,
				SSHSecrets: map[string]string{
					validSecret1Key: validSecret1,
				},
				Groups: []string{groupName1},
			},
		}
		oldObj = &v1alpha1.PaasNS{
			ObjectMeta: metav1.ObjectMeta{
				Name:      paasNsName,
				Namespace: paasName,
			},
			Spec: v1alpha1.PaasNSSpec{
				Paas: paasName,
				SSHSecrets: map[string]string{
					validSecret1Key: validSecret1,
				},
			},
		}
		conf = v1alpha1.PaasConfig{
			Spec: v1alpha1.PaasConfigSpec{
				DecryptKeysSecret: v1alpha1.NamespacedName{
					Name:      paasPkSecret,
					Namespace: paasSystem,
				},
			},
		}
		config.SetConfigV1(conf)
		validator = PaasNSCustomValidator{k8sClient}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
	})

	Context("When properly creating or updating a PaasNS", func() {
		It("Should allow creation", func() {
			By("simulating with reference to Paas where it is deployed")
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().ToNot(HaveOccurred())
		})
		It("Should allow updating", func() {
			By("simulating with reference to Paas where it is deployed")
			obj.Spec.SSHSecrets[validSecret1Key] = validSecret2
			obj.Spec.Groups = []string{groupName1, groupName2}
			warn, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(warn, err).Error().ToNot(HaveOccurred())
		})
	})

	Context("When creating or updating a PaasNS with incorrect Paas reference", func() {
		It("Should deny creation", func() {
			By("created in namespace with wrong prefix")
			obj.Namespace = otherPaasName
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"namespace %s is not named after paas, and not prefixed with 'mypaas-'", otherPaasName))
		})
		It("Should deny creation", func() {
			By("created in ns without Owner Ref")
			obj.Namespace = nsWithoutOwnerRef
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"failed to get owner reference with kind paas and controller=true from namespace resource"))
		})
		It("Should validate paasns name not containing dots", func() {
			obj.Name = "invalid.name.foo"
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn).To(BeNil())
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("paasns name should not contain dots"))
		})
		It("Should validate paasns name", func() {
			for _, test := range []struct {
				name       string
				validation string
				valid      bool
			}{
				{name: "valid-name", validation: "^[a-z-]+$", valid: true},
				{name: "invalid-name", validation: "^[a-z]+$", valid: false},
				{name: "", validation: "^.$", valid: false},
			} {
				conf.Spec.Validations = v1alpha1.PaasConfigValidations{"paasNs": {"name": test.validation}}
				config.SetConfigV1(conf)
				obj.Name = test.name
				if test.valid {
					warn, err := validator.ValidateCreate(ctx, obj)
					Expect(warn).To(BeNil())
					Expect(warn, err).Error().NotTo(HaveOccurred())
				}
			}
		})
	})

	Context("When creating or updating PaasNS with improper group config", func() {
		It("Should deny creation", func() {
			By("creating a PaasNs with improper Group reference")
			obj.Spec.Groups = []string{otherGroup}
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("group %s does not exist in paas", otherGroup))
		})
		It("Should deny updating", func() {
			By("checking references in Paas and raising error if group is not in it")
			obj.Spec.Groups = []string{otherGroup}
			warn, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("group %s does not exist in paas", otherGroup))
		})
	})

	Context("When creating or updating PaasNS with sshSecret that cannot be decrypted", func() {
		It("Should deny creation", func() {
			By("creating PaasNs with sshSecret encrypted with public key that has no corresponding private key")

			var err error
			var invalidSecret1 string
			mycrypt, privateKey, err = newGeneratedCrypt(paasName)
			Expect(err).ToNot(HaveOccurred())
			invalidSecret1, err = mycrypt.Encrypt([]byte("paasns_secret"))
			Expect(err).ToNot(HaveOccurred())
			obj.Spec.SSHSecrets["invalidSecret1"] = invalidSecret1
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to decrypt data with any of the private keys"))
		})
		It("Should deny updating", func() {
			By("creating PaasNs with sshSecret encrypted with public key that has no corresponding private key")

			var err error
			var invalidSecret1 string
			mycrypt, privateKey, err = newGeneratedCrypt(paasName)
			if err != nil {
				Fail(err.Error())
			}
			invalidSecret1, err = mycrypt.Encrypt([]byte("paasns_secret"))
			if err != nil {
				Fail(fmt.Errorf("encrypting invalid paasns secret failed: %w", err).Error())
			}
			obj.Spec.SSHSecrets["invalidSecret1"] = invalidSecret1
			warn, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to decrypt data with any of the private keys"))
		})
	})
	Context("When deleting a PaasNs, ", func() {
		It("Update webhook should not fail", func() {
			By("checking deletion timestamp")

			var err error
			var invalidSecret1 string
			mycrypt, privateKey, err = newGeneratedCrypt(paasName)
			if err != nil {
				Fail(err.Error())
			}
			invalidSecret1, err = mycrypt.Encrypt([]byte("paasns_secret"))
			if err != nil {
				Fail(fmt.Errorf("encrypting invalid paasns secret failed: %w", err).Error())
			}
			obj.Spec.SSHSecrets["invalidSecret1"] = invalidSecret1
			obj.Spec.Paas = "not the one"
			obj.Spec.Groups = []string{otherGroup}
			now := metav1.NewTime(time.Now())
			obj.DeletionTimestamp = &now
			warn, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(warn, err).Error().ToNot(HaveOccurred())
		})
	})
})

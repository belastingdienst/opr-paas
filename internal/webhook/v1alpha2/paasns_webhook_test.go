/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

// Excuse Ginkgo use from revive errors
//revive:disable:dot-imports

import (
	"context"
	"fmt"
	"time"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/config"
	"github.com/belastingdienst/opr-paas/v3/pkg/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("PaasNS Webhook", Ordered, func() {
	const (
		paasSystem        = "paasnssystem"
		paasPkSecret      = "paasns-pk-secret"
		paasName          = "mypaasns-paas"
		paasNameNQ        = "no-quota"
		otherPaasName     = "myotherpaas"
		nsWithoutOwnerRef = paasName + "-without-owner-ref"
		groupName1        = "mygroup1"
		groupName2        = "mygroup2"
		otherGroup        = "myothergroup"
		paasNsName        = "mypaasns"
		paasUID           = paasName + "-uid"
		PaasNQUUID        = paasNameNQ + "-uid"
	)
	var (
		privateKey      []byte
		paas            *v1alpha2.Paas
		noQuotaPaas     *v1alpha2.Paas
		obj             *v1alpha2.PaasNS
		oldObj          *v1alpha2.PaasNS
		mycrypt         *crypt.Crypt
		paasSecret      string
		validSecret1    string
		validSecret1Key = "validSecret1"
		validSecret2    string
		validator       PaasNSCustomValidator
		conf            v1alpha2.PaasConfig
		scheme          *runtime.Scheme
		fakeClient      cl.Client
	)

	BeforeAll(func() {
		var err error

		paas = &v1alpha2.Paas{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Paas",
				APIVersion: "v1alpha2",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
				UID:  paasUID,
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: "paasns-requestor",
				Groups: v1alpha2.PaasGroups{
					groupName1: v1alpha2.PaasGroup{},
					groupName2: v1alpha2.PaasGroup{},
				},
				Secrets: map[string]string{
					paasSecret: paasSecret,
				},
				Quota: quota.Quota{
					"cpu": resource.MustParse("1"),
				},
			},
		}

		noQuotaPaas = &v1alpha2.Paas{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Paas",
				APIVersion: "v1alpha2",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: paasNameNQ,
				UID:  PaasNQUUID,
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: "paasns-requestor",
				Groups: v1alpha2.PaasGroups{
					groupName1: v1alpha2.PaasGroup{},
					groupName2: v1alpha2.PaasGroup{},
				},
				Secrets: map[string]string{
					paasSecret: paasSecret,
				},
				Quota: quota.Quota{},
			},
		}

		scheme = runtime.NewScheme()
		Expect(v1alpha2.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		// Create a fake client that already has the existing Paas
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(paas, noQuotaPaas).
			Build()

		validator = PaasNSCustomValidator{fakeClient}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")

		createNamespace(fakeClient, paasSystem)
		mycrypt, privateKey, err = newGeneratedCrypt(paasName)
		if err != nil {
			Fail(err.Error())
		}
		createPaasPrivateKeySecret(fakeClient, paasSystem, paasPkSecret, privateKey)
		paasSecret, err = mycrypt.Encrypt([]byte("paasSecret"))
		Expect(err).NotTo(HaveOccurred())
		validSecret1, err = mycrypt.Encrypt([]byte("validSecretCreated"))
		Expect(err).NotTo(HaveOccurred())
		validSecret2, err = mycrypt.Encrypt([]byte("validSecretUpdated"))
		Expect(err).NotTo(HaveOccurred())
		createPaasNamespace(fakeClient, *paas, paasName)
		createPaasNamespace(fakeClient, *noQuotaPaas, paasNameNQ)
		// This is a namespace with Owner ref to mypaas and prefix set to myotherpaas-
		createPaasNamespace(fakeClient, *paas, otherPaasName)
		// This is a namespace without Owner ref
		createNamespace(fakeClient, nsWithoutOwnerRef)
	})

	BeforeEach(func() {
		obj = &v1alpha2.PaasNS{
			ObjectMeta: metav1.ObjectMeta{
				Name:      paasNsName,
				Namespace: paasName,
			},
			Spec: v1alpha2.PaasNSSpec{
				Paas: paasName,
				Secrets: map[string]string{
					validSecret1Key: validSecret1,
				},
				Groups: []string{groupName1},
			},
		}
		oldObj = &v1alpha2.PaasNS{
			ObjectMeta: metav1.ObjectMeta{
				Name:      paasNsName,
				Namespace: paasName,
			},
			Spec: v1alpha2.PaasNSSpec{
				Paas: paasName,
				Secrets: map[string]string{
					validSecret1Key: validSecret1,
				},
			},
		}

		conf = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: v1alpha2.PaasConfigSpec{
				DecryptKeysSecret: v1alpha2.NamespacedName{
					Name:      paasPkSecret,
					Namespace: paasSystem,
				},
			},
			Status: v1alpha2.PaasConfigStatus{
				Conditions: []metav1.Condition{
					{
						Type:    v1alpha2.TypeActivePaasConfig,
						Status:  metav1.ConditionTrue,
						Message: "This config is the active config!",
					},
				},
			}}

		err := fakeClient.Create(ctx, &conf)
		Expect(err).To(Not(HaveOccurred()))

		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		err := fakeClient.Delete(ctx, &conf)
		Expect(err).To(Not(HaveOccurred()))
	})

	Context("When properly creating or updating a PaasNS", func() {
		It("Should allow creation", func() {
			By("simulating with reference to Paas where it is deployed")
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().ToNot(HaveOccurred())
		})
		It("Should allow updating", func() {
			By("simulating with reference to Paas where it is deployed")
			obj.Spec.Secrets[validSecret1Key] = validSecret2
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
				"namespace %s is not named after paas, and not prefixed with '%s-'", otherPaasName, paasName))
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
				conf.Spec.Validations = v1alpha2.PaasConfigValidations{"paasNs": {"name": test.validation}}
				ctx = context.WithValue(ctx, config.ContextKeyPaasConfig, conf)

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
			obj.Spec.Secrets["invalidSecret1"] = invalidSecret1
			_, err = validator.ValidateCreate(ctx, obj)
			Expect(err).Error().To(HaveOccurred())
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
			obj.Spec.Secrets["invalidSecret1"] = invalidSecret1
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
			obj.Spec.Secrets["invalidSecret1"] = invalidSecret1
			obj.Spec.Paas = "not the one"
			obj.Spec.Groups = []string{otherGroup}
			now := metav1.NewTime(time.Now())
			obj.DeletionTimestamp = &now
			warn, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(warn, err).Error().ToNot(HaveOccurred())
		})
	})

	Context("When creating a PaasNs but the parent paas doesn't have a quota defined, ", func() {
		It("should not allow creation of a PaasNS", func() {
			By("creating a PaasNS for a Paas without a defined Quota block")

			newObj := obj.DeepCopy()

			newObj.Namespace = "no-quota"
			newObj.Spec.Paas = "no-quota"

			_, err := validator.ValidateCreate(ctx, newObj)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PaasNs cannot be created when there is no quota defined"))
		})
	})
})

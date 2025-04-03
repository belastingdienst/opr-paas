/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"
	"os"
	"time"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/belastingdienst/opr-paas/internal/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createNamespace(ns string) {
	// Create system namespace
	err := k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	})
	if err != nil {
		Fail(fmt.Errorf("failed to create %s namespace: %w", ns, err).Error())
	}
}

func createPaasPrivateKeySecret(ns string, name string, privateKey []byte) {
	// Set up private key
	err := k8sClient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string][]byte{"privatekey0": privateKey},
	})
	if err != nil {
		Fail(fmt.Errorf("failed to create %s.%s secret: %w", ns, name, err).Error())
	}
}

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
		return nil, nil, fmt.Errorf("failed to read private key from file")
	}

	return myCrypt, privateKey, nil
}

var _ = Describe("PaasNS Webhook", Ordered, func() {
	var (
		paasSystem      = "paasnssystem"
		paasPkSecret    = "paasns-pk-secret"
		paasName        = "mypaas"
		otherPaasName   = "myotherpaas"
		groupName1      = "mygroup1"
		groupName2      = "mygroup2"
		otherGroup      = "myothergroup"
		paasNsName      = "mypaasns"
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
		config.SetConfig(conf)
		createNamespace(paasSystem)

		mycrypt, privateKey, err = newGeneratedCrypt(paasName)
		if err != nil {
			Fail(err.Error())
		}

		createPaasPrivateKeySecret(paasSystem, paasPkSecret, privateKey)
		paasSecret, err = mycrypt.Encrypt([]byte("paaSecret"))
		if err != nil {
			Fail(fmt.Errorf("encrypting paas secret failed: %w", err).Error())
		}
		validSecret1, err = mycrypt.Encrypt([]byte("validSecretCreated"))
		if err != nil {
			Fail(fmt.Errorf("encrypting valid paasns secret failed: %w", err).Error())
		}
		validSecret2, err = mycrypt.Encrypt([]byte("validSecretUpdated"))
		if err != nil {
			Fail(fmt.Errorf("encrypting valid paasns secret failed: %w", err).Error())
		}
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
		if err != nil {
			Fail(fmt.Errorf("failed to create paas: %w", err).Error())
		}
		createNamespace(paasName)
		createNamespace(otherPaasName)
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
		config.SetConfig(conf)
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
			By("simulating with reference to other Paas then where created")
			obj.Spec.Paas = otherPaasName
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("paas %s does not exist", otherPaasName))
		})
		It("Should deny creation", func() {
			By("simulating PaasNs creation in Namespace for other Paas")
			obj.Namespace = otherPaasName
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("paasns not in namespace belonging to paas %s", paasName))
		})
		It("Should deny updating", func() {
			By("checking old vs new and raising error if they differ")
			obj.Spec.Paas = "changing this should fail"
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err.Error()).To(ContainSubstring("field is immutable"))
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
				config.SetConfig(conf)
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
			if err != nil {
				Fail(err.Error())
			}
			invalidSecret1, err = mycrypt.Encrypt([]byte("paasns_secret"))
			if err != nil {
				Fail(fmt.Errorf("encrypting invalid paasns secret failed: %w", err).Error())
			}
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

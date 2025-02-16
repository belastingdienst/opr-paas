/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"
	"os"

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
	privateKeyFile, err := os.CreateTemp("", "private")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get new tmp private key file")
	}
	publicKeyFile, err := os.CreateTemp("", "public")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get new tmp private key file")
	}
	myCrypt, err = crypt.NewGeneratedCrypt(privateKeyFile.Name(), publicKeyFile.Name(), context)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get new tmp private key file")
	}
	privateKey, err = os.ReadFile(privateKeyFile.Name())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key from file")
	}

	return myCrypt, privateKey, nil
}

var _ = Describe("PaasNS Webhook", Ordered, func() {
	var (
		paasSystem    string = "paasnssystem"
		paasPkSecret  string = "paasns-pk-secret"
		paasName      string = "mypaas"
		otherPaasName string = "myotherpaas"
		groupName     string = "mygroup"
		otherGroup    string = "myothergroup"
		paasNsName    string = "mypaasns"
		privateKey    []byte
		paas          *v1alpha1.Paas
		obj           *v1alpha1.PaasNS
		oldObj        *v1alpha1.PaasNS
		mycrypt       *crypt.Crypt
		paasSecret    string
		validSecret   string
		validator     PaasNSCustomValidator
	)

	BeforeAll(func() {
		var err error

		conf := v1alpha1.PaasConfig{
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
		validSecret, err = mycrypt.Encrypt([]byte("validSecret"))
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
					groupName: v1alpha1.PaasGroup{},
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
					"validSecret": validSecret,
				},
			},
		}
		oldObj = &v1alpha1.PaasNS{}
		validator = PaasNSCustomValidator{k8sClient}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
	})

	Context("When creating a proper PaasNS", func() {
		It("Should allow creation", func() {
			By("simulating with reference to Paas where it is deployed")
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().ToNot(HaveOccurred())
		})
	})

	Context("When creating a PaasNS with incorrect Paas reference", func() {
		It("Should deny creation", func() {
			By("simulating with reference to other Paas then where created")
			obj.Spec.Paas = otherPaasName
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("paas %s does not exist", otherPaasName))
		})
		It("Should deny creation", func() {
			By("simulating PaasNs creation in Namespace for other Paas")
			obj.ObjectMeta.Namespace = otherPaasName
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("paasns not in namespace belonging to paas %s", paasName))
		})
	})

	Context("When creating PaasNS with improper group config", func() {
		It("Should deny creation", func() {
			By("creating a PaasNs with improper Group reference")
			obj.Spec.Groups = []string{otherGroup}
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("group %s does not exist in paas", otherGroup))
		})
	})

	Context("When creating PaasNS with sshSecret that cannot be decrypted", func() {
		It("Should deny creation", func() {
			By("creating PaasNs with sshSecret encrypted with public key that has no corresponding private key")

			var err error
			var invalidSecret string
			mycrypt, privateKey, err = newGeneratedCrypt(paasName)
			if err != nil {
				Fail(err.Error())
			}
			invalidSecret, err = mycrypt.Encrypt([]byte("paasns_secret"))
			if err != nil {
				Fail(fmt.Errorf("encrypting invalid paasns secret failed: %w", err).Error())
			}
			obj.Spec.SSHSecrets["invalidSecret"] = invalidSecret
			warn, err := validator.ValidateCreate(ctx, obj)
			Expect(warn, err).Error().To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to decrypt data with any of the private keys"))
		})
	})
})

/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	paasquota "github.com/belastingdienst/opr-paas/internal/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("testing hashdata", func() {
	When("hashing a string", func() {
		It("should not return an error", func() {
			for _, test := range []struct {
				input    string
				expected string
			}{
				{
					input: "My Wonderful Test String",
					// revive:disable-next-line
					expected: "703fe1668c39ec0fdf3c9916d526ba4461fe10fd36bac1e2a1b708eb8a593e418eb3f92dbbd2a6e3776516b0e03743a45cfd69de6a3280afaa90f43fa1918f74",
				},
				{
					input: "Another Wonderful Test String",
					// revive:disable-next-line
					expected: "d3bfd910013886fe68ffd5c5d854e7cb2a8ce2a15a48ade41505b52ce7898f63d8e6b9c84eacdec33c45f7a2812d93732b524be91286de328bbd6b72d5aee9de",
				},
			} {
				Expect(hashData(test.input)).To(Equal(test.expected))
			}
		})
	})
})

var _ = Describe("secret controller", Ordered, func() {
	const (
		paasRequestor      = "paas-controller"
		paasName           = "my-paas"
		capAppSetNamespace = "asns"
		capAppSetName      = "argoas"
		capName            = "argocd"
		paasSystem         = "paasnssystem"
		paasPkSecret       = "secret-pk-secret"
	)
	var (
		paas       *api.Paas
		reconciler *PaasReconciler
		myConfig   api.PaasConfig
		privateKey []byte
		mycrypt    *crypt.Crypt
		pns        *api.PaasNS
	)
	ctx := context.Background()

	BeforeAll(func() {
		var err error

		assureNamespace(ctx, paasName)
		assureNamespace(ctx, paasSystem)
		mycrypt, privateKey, err = newGeneratedCrypt(paasRequestor)
		Expect(err).NotTo(HaveOccurred())

		createPaasPrivateKeySecret(paasSystem, paasPkSecret, privateKey)

		encrypted, err := mycrypt.Encrypt([]byte("some encrypted string"))
		Expect(err).NotTo(HaveOccurred())

		pns = &api.PaasNS{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: paasName},
			Spec: api.PaasNSSpec{
				Paas: paasName,
				SSHSecrets: map[string]string{
					"probably a git repo.git": encrypted, // already base64 encoded by crypt.Encrypt
				},
			},
		}
	})

	BeforeEach(func() {
		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		paas = &api.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasRequestor,
			},
			Spec: api.PaasSpec{
				Requestor: paasRequestor,
				Capabilities: api.PaasCapabilities{
					capName: api.PaasCapability{
						Enabled: true,
						// TODO: For next tests
						// SSHSecrets: map[string]string{},
					},
				},
				Quota: paasquota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
				// TODO: For next tests
				// Namespaces: []string{"my-namespace"},
				// SSHSecrets: map[string]string{},
			},
		}

		// Delete if exists to avoid "already exists" error
		// We ignore the error because it might not exist
		_ = k8sClient.Delete(ctx, &api.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasRequestor,
			},
		})

		// Create the Paas in the cluster to get a UID
		err := k8sClient.Create(ctx, paas)
		Expect(err).NotTo(HaveOccurred())

		myConfig = api.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: api.PaasConfigSpec{
				ClusterWideArgoCDNamespace: capAppSetNamespace,
				Capabilities: map[string]api.ConfigCapability{
					capName: {
						AppSet: capAppSetName,
						QuotaSettings: api.ConfigQuotaSettings{
							DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
								corev1.ResourceLimitsCPU: resourcev1.MustParse("5"),
							},
						},
					},
				},
				Debug: false,
				DecryptKeysSecret: api.NamespacedName{
					Name:      paasPkSecret,
					Namespace: paasSystem,
				},
				ManagedByLabel:  "argocd.argoproj.io/manby",
				ManagedBySuffix: "argocd",
				RequestorLabel:  "o.lbl",
				QuotaLabel:      "q.lbl",
				GroupSyncList: api.NamespacedName{
					Namespace: "gsns",
					Name:      "wlname",
				},
				GroupSyncListKey: "groupsynclist.txt",
			},
		}
		config.SetConfig(myConfig)
	})

	When("reconciling a PaasNS with a SshSecrets value", func() {
		It("should not return an error", func() {
			err := reconciler.reconcileNamespaceSecrets(ctx, paas, pns, pns.GetObjectMeta().GetNamespace(), pns.Spec.SSHSecrets)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a secret with the decrypted data", func() {
			secrets := &corev1.SecretList{}
			err := k8sClient.List(ctx, secrets, client.InNamespace(paasName))
			Expect(err).NotTo(HaveOccurred())

			Expect(secrets.Items).To(HaveLen(1))
			Expect(secrets.Items[0].Data["url"]).To(Equal([]byte("probably a git repo.git")))
			Expect(secrets.Items[0].Data["sshPrivateKey"]).To(Equal([]byte("some encrypted string")))
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

func createPaasPrivateKeySecret(ns string, name string, privateKey []byte) {
	ctx := context.TODO()
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

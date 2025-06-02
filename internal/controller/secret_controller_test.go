/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/api/v1alpha2"
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
		paasRequestor      = "paas-controller-test"
		paasName           = "secret-controller-paas"
		capAppSetNamespace = "asns"
		capAppSetName      = "argoas"
		capName            = "argocd"
		paasSystem         = "paasnssystem"
		paasPkSecret       = "secret-pk-secret"
	)
	var (
		paas            *v1alpha2.Paas
		reconciler      *PaasReconciler
		myConfig        v1alpha2.PaasConfig
		privateKey      []byte
		mycrypt         *crypt.Crypt
		pns             *v1alpha2.PaasNS
		encryptedString string
	)
	ctx := context.Background()

	BeforeAll(func() {
		var err error

		assureNamespace(ctx, paasName)
		assureNamespace(ctx, paasSystem)
		mycrypt, privateKey, err = newGeneratedCrypt(paasRequestor)
		Expect(err).NotTo(HaveOccurred())

		createPaasPrivateKeySecret(ctx, paasSystem, paasPkSecret, privateKey)

		encryptedString, err = mycrypt.Encrypt([]byte("some encrypted string"))
		Expect(err).NotTo(HaveOccurred())

		pns = &v1alpha2.PaasNS{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: paasName},
			Spec: v1alpha2.PaasNSSpec{
				Paas: paasName,
				Secrets: map[string]string{
					"paasns-git-repo": encryptedString,
				},
			},
		}
	})

	BeforeEach(func() {
		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		paas = &v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasRequestor,
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: paasRequestor,
				Capabilities: v1alpha2.PaasCapabilities{
					capName: v1alpha2.PaasCapability{
						Secrets: map[string]string{
							"paas-capability-git-repo": encryptedString,
						},
					},
				},
				Quota: paasquota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
				Namespaces: v1alpha2.PaasNamespaces{
					paasName: v1alpha2.PaasNamespace{},
				},
				Secrets: map[string]string{
					"paas-namespace-git-repo": encryptedString,
				},
			},
		}

		// Delete if exists to avoid "already exists" error
		_ = k8sClient.Delete(ctx, &api.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasRequestor,
			},
		})

		// Create the Paas in the cluster to get a UID
		err := k8sClient.Create(ctx, paas)
		Expect(err).NotTo(HaveOccurred())

		myConfig = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: v1alpha2.PaasConfigSpec{
				ClusterWideArgoCDNamespace: capAppSetNamespace,
				Capabilities: map[string]v1alpha2.ConfigCapability{
					capName: {
						AppSet: capAppSetName,
						QuotaSettings: v1alpha2.ConfigQuotaSettings{
							DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
								corev1.ResourceLimitsCPU: resourcev1.MustParse("5"),
							},
						},
					},
				},
				Debug: false,
				DecryptKeysSecret: v1alpha2.NamespacedName{
					Name:      paasPkSecret,
					Namespace: paasSystem,
				},
				ManagedByLabel:  "argocd.argoproj.io/manby",
				ManagedBySuffix: "argocd",
				RequestorLabel:  "o.lbl",
				QuotaLabel:      "q.lbl",
			},
		}
		config.SetConfig(myConfig)
	})

	When("reconciling a PaasNS with a SshSecrets value", func() {
		It("should not return an error", func() {
			err := reconciler.reconcileNamespaceSecrets(ctx, paas, pns, pns.GetObjectMeta().GetNamespace(),
				pns.Spec.Secrets)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a secret with the decrypted data", func() {
			secrets := &corev1.SecretList{}
			err := k8sClient.List(ctx, secrets, client.InNamespace(paasName))
			Expect(err).NotTo(HaveOccurred())

			found := findSecretByURL(secrets.Items, "paasns-git-repo")
			Expect(found).NotTo(BeNil())
			Expect(found.Data["sshPrivateKey"]).To(Equal([]byte("some encrypted string")))
		})
	})

	When("reconciling a paas namespace with a SshSecrets value", func() {
		It("should not return an error", func() {
			err := reconciler.reconcileNamespaceSecrets(ctx, paas, pns, paasName,
				paas.Spec.Secrets)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a secret with the decrypted data", func() {
			secrets := &corev1.SecretList{}
			err := k8sClient.List(ctx, secrets, client.InNamespace(paasName))
			Expect(err).NotTo(HaveOccurred())

			found := findSecretByURL(secrets.Items, "paas-namespace-git-repo")
			Expect(found).NotTo(BeNil())
			Expect(found.Data["sshPrivateKey"]).To(Equal([]byte("some encrypted string")))
		})
	})

	When("reconciling a paas capability with a SSHSecret", func() {
		It("should not return an error", func() {
			err := reconciler.reconcileNamespaceSecrets(ctx, paas, pns, paasName,
				paas.Spec.Capabilities[capName].Secrets)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a secret with the decrypted data", func() {
			secrets := &corev1.SecretList{}
			err := k8sClient.List(ctx, secrets, client.InNamespace(paasName))
			Expect(err).NotTo(HaveOccurred())

			found := findSecretByURL(secrets.Items, "paas-capability-git-repo")
			Expect(found).NotTo(BeNil())
			Expect(found.Data["sshPrivateKey"]).To(Equal([]byte("some encrypted string")))
		})
	})

	When("reconciling a paas namespace with one secret removed", func() {
		It("should not return an error", func() {
			err := reconciler.reconcileNamespaceSecrets(ctx, paas, pns, pns.GetObjectMeta().GetNamespace(),
				paas.Spec.Capabilities[capName].Secrets)
			Expect(err).NotTo(HaveOccurred())

			// Remove the secret from the paas spec (simulate user removing the secret)
			capability := paas.Spec.Capabilities[capName]
			capability.Secrets = nil
			paas.Spec.Capabilities[capName] = capability
			err = k8sClient.Update(ctx, paas)
			Expect(err).NotTo(HaveOccurred())

			// Reconcile again with SSHSecrets now nil (should trigger deletion)
			err = reconciler.reconcileNamespaceSecrets(ctx, paas, pns, pns.GetObjectMeta().GetNamespace(),
				paas.Spec.Capabilities[capName].Secrets)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have removed this secret", func() {
			secrets := &corev1.SecretList{}
			err := k8sClient.List(ctx, secrets, client.InNamespace(paasName))
			Expect(err).NotTo(HaveOccurred())

			found := findSecretByURL(secrets.Items, "paas-capability-git-repo")
			Expect(found).To(BeNil())
		})
	})
})

func findSecretByURL(secrets []corev1.Secret, url string) *corev1.Secret {
	for _, s := range secrets {
		if string(s.Data["url"]) == url {
			return &s
		}
	}
	return nil
}

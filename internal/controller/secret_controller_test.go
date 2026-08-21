/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"
	"maps"

	"github.com/belastingdienst/opr-paas-cli/v2/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v5/internal/config"
	"github.com/belastingdienst/opr-paas/v5/internal/utils"
	paasquota "github.com/belastingdienst/opr-paas/v5/pkg/quota"
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

var _ = Describe("testing mergeSecrets", func() {
	When("merging secrets", func() {
		It("should not return an error", func() {
			for _, tt := range []struct {
				name     string
				base     map[string]string
				override map[string]string
				want     map[string]string
			}{
				{
					name:     "empty base and override",
					base:     map[string]string{},
					override: map[string]string{},
					want:     map[string]string{},
				},
				{
					name:     "base only",
					base:     map[string]string{"a1": "1"},
					override: map[string]string{},
					want:     map[string]string{"a1": "1"},
				},
				{
					name:     "override only",
					base:     map[string]string{},
					override: map[string]string{"b": "b2"},
					want:     map[string]string{"b": "b2"},
				},
				{
					name:     "override replaces base",
					base:     map[string]string{"c": "c1"},
					override: map[string]string{"c": "c2"},
					want:     map[string]string{"c": "c2"},
				},
				{
					name:     "override adds to base",
					base:     map[string]string{"a": "1"},
					override: map[string]string{"b": "2"},
					want:     map[string]string{"a": "1", "b": "2"},
				},
				{
					name:     "multiple overrides",
					base:     map[string]string{"f": "1", "c": "3"},
					override: map[string]string{"f": "10", "g": "20"},
					want:     map[string]string{"f": "10", "g": "20", "c": "3"},
				},
			} {
				// copy maps to avoid mutating original test cases
				baseCopy := maps.Clone(tt.base)
				overrideCopy := maps.Clone(tt.override)

				got := mergeSecrets(baseCopy, overrideCopy)
				Expect(got).To(Equal(tt.want))
			}

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
		paasRequestor      = "secret-controller-requestor"
		paasName           = "secret-controller-paas"
		nsName             = "ns1"
		pnsName            = "pns1"
		capAppSetNamespace = "asns"
		capAppSetName      = "argoas"
		capName            = "argocd"
		paasSystem         = "paasnssystem"
		paasPkSecret       = "secret-pk-secret"
		capSecretURL       = "paas-capability-git-repo"
		pnsSecretURL       = "paasns-git-repo"
		nsSecretURL        = "paas-namespace-git-repo"
		decryptedValue     = "some encrypted string"
		template           = ""
		simpleTemplate     = `{{ $s := dict }}{{ $_ := set $s "scrt" .Paas.Spec.Capabilities.mycap.Secrets }}{{ $s }}`
		argoTemplate       = `{{ $s := dict }}{{ $_ := set $s "scrt" .Paas.Spec.Capabilities.mycap.Secrets }}{{ $s }}`
		tektonTemplate     = `{{ $s := dict }}{{ $_ := set $s "scrt" .Paas.Spec.Capabilities.mycap.Secrets }}{{ $s }}`
	)
	var (
		paas            *v1alpha2.Paas
		reconciler      *PaasReconciler
		myConfig        v1alpha2.PaasConfig
		privateKey      []byte
		mycrypt         *crypt.Crypt
		pns             *v1alpha2.PaasNS
		encryptedString string
		nsNamespace     = utils.Join(paasName, nsName)
		pnsNamespace    = utils.Join(paasName, pnsName)
		capNamespace    = utils.Join(paasName, capName)
	)
	ctx := context.Background()

	BeforeAll(func() {
		var err error

		assureNamespace(ctx, paasSystem)
		mycrypt, privateKey, err = newGeneratedCrypt(paasName)
		Expect(err).NotTo(HaveOccurred())

		createPaasPrivateKeySecret(ctx, paasSystem, paasPkSecret, privateKey)

		encryptedString, err = mycrypt.Encrypt([]byte(decryptedValue))
		Expect(err).NotTo(HaveOccurred())

		pns = &v1alpha2.PaasNS{
			ObjectMeta: metav1.ObjectMeta{Name: pnsName, Namespace: nsNamespace},
			Spec: v1alpha2.PaasNSSpec{
				Secrets: map[string]string{
					pnsSecretURL: encryptedString,
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
				Name: paasName,
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: paasRequestor,
				Capabilities: v1alpha2.PaasCapabilities{
					capName: v1alpha2.PaasCapability{
						Secrets: map[string]string{
							capSecretURL: encryptedString,
						},
					},
				},
				Quota: paasquota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
				Namespaces: v1alpha2.PaasNamespaces{
					nsName: v1alpha2.PaasNamespace{},
				},
				Secrets: map[string]string{
					nsSecretURL: encryptedString,
				},
			},
		}

		// Delete if exists to avoid "already exists" error
		_ = k8sClient.Delete(ctx, &v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
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
						Secrets: simpleTemplate,
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

		// Updates context to include paasConfig
		ctx = context.WithValue(context.Background(), config.ContextKeyPaasConfig, myConfig)
	})

	When("reconciling a PaasNS with a Secrets value", func() {
		It("should not return an error", func() {
			err := reconciler.reconcileNamespaceSecrets(ctx, paas, pns, pnsNamespace, pns.Spec.Secrets)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a secret with the decrypted data", func() {
			Expect(verifySecret(ctx, pnsNamespace, "scrt", map[string]string{capSecretURL: decryptedValue})).
				NotTo(HaveOccurred())
		})
	})

	When("reconciling a paas namespace with a Secrets value", func() {
		It("should not return an error", func() {
			err := reconciler.reconcileNamespaceSecrets(ctx, paas, nil, nsNamespace, paas.Spec.Secrets)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a secret with the decrypted data", func() {
			Expect(verifySecret(ctx, nsNamespace, "scrt", map[string]string{capSecretURL: decryptedValue})).
				NotTo(HaveOccurred())
		})
	})

	When("reconciling a paas capability with a Secret", func() {
		It("should not return an error", func() {
			err := reconciler.reconcileNamespaceSecrets(ctx, paas, nil, capName,
				paas.Spec.Capabilities[capName].Secrets)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a secret with the decrypted data", func() {
			Expect(verifySecret(ctx, capNamespace, "scrt", map[string]string{capSecretURL: decryptedValue})).
				NotTo(HaveOccurred())
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
			Expect(
				verifySecret(ctx, pns.GetNamespace(), "scrt", map[string]string{capSecretURL: decryptedValue}).
					Error()).
				To(HavePrefix("could not find secret"))
		})
	})
	When("reconciling for different caps", func() {
		var tests = []struct {
			template string
			secrets  map[string]map[string]string
		}{
			{template: simpleTemplate, secrets: map[string]map[string]string{"": {"type": "Z2l0"}}},
			{template: argoTemplate, secrets: map[string]map[string]string{"": {"type": "Z2l0"}}},
			{template: tektonTemplate, secrets: map[string]map[string]string{"": {"type": "Z2l0"}}},
		}
		It("should be able to create different types of cap secrets", func() {
			for _, test := range tests {
				fmt.Fprintf(GinkgoWriter, "DEBUG - Test: %v", test)
				for secretName, secretData := range test.secrets {
					Expect(verifySecret(ctx, pns.GetNamespace(), secretName, secretData)).NotTo(HaveOccurred())
				}
			}
		})
		It("should be able to create different types of ns secrets", func() {
			for _, test := range tests {
				fmt.Fprintf(GinkgoWriter, "DEBUG - Test: %v", test)
				for secretName, secretData := range test.secrets {
					Expect(verifySecret(ctx, pns.GetNamespace(), secretName, secretData)).NotTo(HaveOccurred())
				}
			}
		})
	})
})

func verifySecret(ctx context.Context, ns string, name string, data map[string]string) error {
	secret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, secret); err != nil {
		return err
	}
	for key, checkValue := range data {
		returnedValue, exists := secret.Data[key]
		if !exists {
			return fmt.Errorf("Cannot find %s in secret data %v", key, secret)
		}
		if checkValue != string(returnedValue) {
			return fmt.Errorf("%s is not %s for key %s in secret %s", checkValue, string(returnedValue), key, secret)
		}
	}
	return nil
}

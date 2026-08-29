/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"

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
		capSecretName      = "paas-capability-secret"
		pnsSecretName      = "paasns-secret"
		nsSecretName       = "paas-namespace-secret"
		secretName         = "paas-secret"
		decryptedValue     = "user:password"
		template           = ""
		simpleTemplate     = `
          {{- $paasSecrets := getPaasSecrets -}}
          {{- if gt (len $paasSecrets) 0 -}}
            {{- $result := dict "paas-secrets" $paasSecrets -}}
            {{- $result | toYAML -}}
          {{- end -}}`
		tektonTemplate = `
          {{- $auths := dict -}}
          {{- range $name, $decrypted := getPaasSecrets -}}
            {{- $auth := base64Encode $decrypted -}}
            {{- $parts := split ":" $decrypted -}}
            {{- $username := index $parts "_0" -}}
            {{- $password := index $parts "_1" -}}
            {{- $authData := dict "username" $username "password" $password "auth" $auth -}}
            {{- $_ := $auths | set $name $authData -}}
          {{- end -}}
          {{- $dockerConfigJSON := toJSON (dict "auths" $auths) -}}
          {{- $secrets := dict "container-repo-token" (dict ".dockerconfigjson" $dockerConfigJSON) -}}
          {{ $secrets | toYAML }}`
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
		capConfig       = v1alpha2.ConfigCapability{
			AppSet: capAppSetName,
			QuotaSettings: v1alpha2.ConfigQuotaSettings{
				DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
					corev1.ResourceLimitsCPU: resourcev1.MustParse("5"),
				},
			},
			Secrets: simpleTemplate,
		}
	)
	ctx := context.Background()

	BeforeAll(func() {
		var err error

		for _, ns := range []string{paasSystem, nsNamespace, pnsNamespace, capNamespace} {
			assureNamespace(ctx, ns)
		}
		mycrypt, privateKey, err = newGeneratedCrypt(paasName)
		Expect(err).NotTo(HaveOccurred())

		createPaasPrivateKeySecret(ctx, paasSystem, paasPkSecret, privateKey)

		encryptedString, err = mycrypt.Encrypt([]byte(decryptedValue))
		Expect(err).NotTo(HaveOccurred())

		pns = &v1alpha2.PaasNS{
			ObjectMeta: metav1.ObjectMeta{Name: pnsName, Namespace: nsNamespace},
			Spec: v1alpha2.PaasNSSpec{
				Secrets: map[string]string{
					pnsSecretName: encryptedString,
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
					capName: v1alpha2.PaasCapability{Secrets: map[string]string{capSecretName: encryptedString}}},
				Quota: paasquota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
				Namespaces: v1alpha2.PaasNamespaces{
					nsName: v1alpha2.PaasNamespace{
						Secrets: map[string]string{nsSecretName: encryptedString},
					},
				},
				Secrets: map[string]string{
					nsSecretName: encryptedString,
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
					capName: capConfig,
				},
				Debug: false,
				DecryptKeysSecret: v1alpha2.NamespacedName{
					Name:      paasPkSecret,
					Namespace: paasSystem,
				},
				ManagedByLabel:   "argocd.argoproj.io/manby",
				ManagedBySuffix:  "argocd",
				RequestorLabel:   "o.lbl",
				QuotaLabel:       "q.lbl",
				NamespaceSecrets: simpleTemplate,
			},
		}

		// Updates context to include paasConfig
		ctx = context.WithValue(context.Background(), config.ContextKeyPaasConfig, myConfig)
	})

	When("reconciling a PaasNS with a Secrets value", func() {
		It("should not return an error", func() {
			nsDefs := namespaceDefs{
				pns.Name: namespaceDef{
					nsName:           pnsNamespace,
					paasns:           pns,
					quotaName:        pnsNamespace,
					encryptedSecrets: pns.Spec.Secrets,
				},
			}
			err := reconciler.reconcilePaasSecrets(ctx, paas, nsDefs)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a secret with the decrypted data", func() {
			secrets, err := listSecrets(ctx, pnsNamespace)
			Expect(err).NotTo(HaveOccurred())
			fmt.Fprintf(GinkgoWriter, "DEBUG - secrets in namespace %s: %v", pnsNamespace, secrets)

			Expect(verifySecret(ctx, pnsNamespace, "paas-secrets", map[string]string{pnsSecretName: decryptedValue})).
				NotTo(HaveOccurred())
		})
	})

	When("reconciling a paas namespace with a Secrets value", func() {
		It("should not return an error", func() {
			nsDefs := namespaceDefs{
				pns.Name: namespaceDef{
					nsName:           nsNamespace,
					quotaName:        nsNamespace,
					encryptedSecrets: paas.Spec.Namespaces[nsName].Secrets,
				},
			}
			err := reconciler.reconcilePaasSecrets(ctx, paas, nsDefs)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a secret with the decrypted data", func() {
			Expect(verifySecret(ctx, nsNamespace, "paas-secrets", map[string]string{nsSecretName: decryptedValue})).
				NotTo(HaveOccurred())
		})
	})

	When("reconciling a paas capability with a Secret", func() {
		It("should not return an error", func() {
			nsDefs := namespaceDefs{
				pns.Name: namespaceDef{
					capName:          capName,
					nsName:           capNamespace,
					quotaName:        capNamespace,
					encryptedSecrets: paas.Spec.Capabilities[capName].Secrets,
				},
			}
			err := reconciler.reconcilePaasSecrets(ctx, paas, nsDefs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a secret with the decrypted data", func() {
			Expect(verifySecret(ctx, capNamespace, "paas-secrets", map[string]string{capSecretName: decryptedValue})).
				NotTo(HaveOccurred())
		})
	})

	When("reconciling a paas namespace with one secret removed", func() {
		It("should not return an error", func() {
			nsDefs := namespaceDefs{
				pns.Name: namespaceDef{
					nsName:           nsNamespace,
					quotaName:        nsNamespace,
					encryptedSecrets: paas.Spec.Secrets,
				},
			}
			err := reconciler.reconcilePaasSecrets(ctx, paas, nsDefs)
			Expect(err).NotTo(HaveOccurred())

			// Remove the secret from the paas spec (simulate user removing the secret)
			paas.Spec.Secrets = nil
			err = k8sClient.Update(ctx, paas)
			Expect(err).NotTo(HaveOccurred())

			// Reconcile again with SSHSecrets now nil (should trigger deletion)
			nsDefs[pns.Name] = namespaceDef{
				nsName:           nsNamespace,
				quotaName:        nsNamespace,
				encryptedSecrets: nil,
			}
			err = reconciler.reconcilePaasSecrets(ctx, paas, nsDefs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have removed this secret", func() {
			secrets, err := listSecrets(ctx, nsNamespace)
			Expect(err).NotTo(HaveOccurred())
			fmt.Fprintf(GinkgoWriter, "DEBUG - secrets in namespace %s: %v\n", nsNamespace, secrets)
			Expect(
				verifySecret(ctx, nsNamespace, "paas-secrets", map[string]string{nsSecretName: decryptedValue}).
					Error()).To(HavePrefix(`secrets "paas-secrets" not found`))
		})
	})
	When("reconciling for different caps", func() {
		var tests = []struct {
			template string
			secrets  map[string]map[string]string
		}{
			{template: simpleTemplate, secrets: map[string]map[string]string{"paas-secrets": {capSecretName: decryptedValue}}},
			{template: backwardsCompatibleTemplate, secrets: map[string]map[string]string{
				hashedSecretName(capSecretName): {
					"type":          "git",
					"url":           capSecretName,
					"sshPrivateKey": decryptedValue,
				}}},
			{template: tektonTemplate, secrets: map[string]map[string]string{"container-repo-token": {
				// revive:disable-next-line
				".dockerconfigjson": `{"auths":{"paas-capability-secret":{"auth":"dXNlcjpwYXNzd29yZA==","password":"password","username":"user"}}}`,
			}}},
		}
		It("should be able to create different types of cap secrets", func() {
			nsDef := namespaceDef{
				capName:          capName,
				nsName:           capNamespace,
				quotaName:        capNamespace,
				encryptedSecrets: paas.Spec.Capabilities[capName].Secrets,
			}
			for _, test := range tests {
				// Display test
				fmt.Fprintf(GinkgoWriter, "DEBUG - Test: %v\n", test)

				// Add template to config
				capConfig.Secrets = test.template
				myConfig.Spec.Capabilities[capName] = capConfig
				ctx = context.WithValue(context.Background(), config.ContextKeyPaasConfig, myConfig)
				nsDefs := namespaceDefs{pns.Name: nsDef}

				// Run reconcilePaasSecrets for the cap with the template
				err := reconciler.reconcilePaasSecrets(ctx, paas, nsDefs)
				Expect(err).NotTo(HaveOccurred())

				// check for secrets
				secrets, err := listSecrets(ctx, capNamespace)
				Expect(err).NotTo(HaveOccurred())
				fmt.Fprintf(GinkgoWriter, "DEBUG - secrets in namespace %s: %v\n", capNamespace, secrets)

				// Check that secret exists and looks as expected
				for secretName, secretData := range test.secrets {
					Expect(verifySecret(ctx, capNamespace, secretName, secretData)).NotTo(HaveOccurred())
				}
			}
		})
		It("should be able to create different types of ns secrets", func() {
			nsDefs := namespaceDefs{
				pns.Name: namespaceDef{
					nsName:           nsNamespace,
					quotaName:        nsNamespace,
					encryptedSecrets: paas.Spec.Capabilities[capName].Secrets,
				}}
			for _, test := range tests {
				// Print test details
				fmt.Fprintf(GinkgoWriter, "DEBUG - Test: %v\n", test)

				// Add template to config
				myConfig.Spec.NamespaceSecrets = test.template
				ctx = context.WithValue(context.Background(), config.ContextKeyPaasConfig, myConfig)

				// Run reconcilePaasSecrets for the ns
				err := reconciler.reconcilePaasSecrets(ctx, paas, nsDefs)
				Expect(err).NotTo(HaveOccurred())

				// check for secrets
				secrets, err := listSecrets(ctx, nsNamespace)
				Expect(err).NotTo(HaveOccurred())
				fmt.Fprintf(GinkgoWriter, "DEBUG - secrets in namespace %s: %v\n", nsNamespace, secrets)

				// Check that secret exists and looks as expected
				for secretName, secretData := range test.secrets {
					Expect(verifySecret(ctx, nsNamespace, secretName, secretData)).NotTo(HaveOccurred())
				}
			}
		})
	})
})

func verifySecret(ctx context.Context, ns string, secretName string, secretData map[string]string) error {
	secret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: secretName}, secret); err != nil {
		return err
	}
	for key, checkValue := range secretData {
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

func listSecrets(ctx context.Context, ns string) ([]string, error) {
	secrets := &corev1.SecretList{}
	listOpts := []client.ListOption{
		client.InNamespace(ns),
	}
	if err := k8sClient.List(ctx, secrets, listOpts...); err != nil {
		return nil, err
	}
	var secretNames []string
	for _, secret := range secrets.Items {
		secretNames = append(secretNames, secret.Name)
	}
	return secretNames, nil
}

/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

/*
var _ = describe("secret controller", ordered, func() {
	ctx := context.background()

	beforeall(func() {
		// set the paasconfig so reconcilers know where to find our fixtures
		config.setconfig(genericconfig)
	})

	var reconciler *paasreconciler
	beforeeach(func() {
		reconciler = &paasreconciler{
			client: k8sclient,
			scheme: k8sclient.scheme(),
		}
	})

	when("reconciling a paasns with no secrets", func() {
		pns := &api.paasns{
			objectmeta: metav1.objectmeta{name: "foo"},
			spec: api.paasnsspec{
				paas: "my-paas",
			},
		}

		it("should not return an error", func() {
			err := reconciler.reconcilesecret(ctx, &api.paas{}, pns)

			expect(err).notto(haveoccurred())
		})

		it("should not create any secrets", func() {
			secrets := &corev1.secretlist{}
			err := k8sclient.list(ctx, secrets, client.innamespace("my-paas-foo"))

			expect(err).notto(haveoccurred())
			expect(secrets.items).to(bezero())
		})
	})

	When("reconciling a PaasNS with an SshSecrets value", func() {
		paas := &api.Paas{ObjectMeta: metav1.ObjectMeta{
			Name: "my-paas",
			UID:  "abc", // Needed or owner references fail
		}}
		var pns *api.PaasNS
		BeforeAll(func() {
			encrypted, err := rsa.EncryptOAEP(
				sha512.New(),
				rand.Reader,
				pubkey,
				[]byte("some encrypted string"),
				[]byte("my-paas"),
			)
			Expect(err).NotTo(HaveOccurred())

			pns = &api.PaasNS{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "my-paas"},
				Spec: api.PaasNSSpec{
					Paas: "my-paas",
					SSHSecrets: map[string]string{
						"probably a git repo.git": base64.StdEncoding.EncodeToString(encrypted),
					},
				},
			}
			err = k8sClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "my-paas"},
			})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "my-paas-foo"},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not return an error", func() {
			err := reconciler.reconcileSecret(ctx, paas, pns)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a secret with the decrypted data", func() {
			secrets := &corev1.SecretList{}
			err := k8sClient.List(ctx, secrets, client.InNamespace("my-paas-foo"))
			Expect(err).NotTo(HaveOccurred())

			Expect(secrets.Items).To(HaveLen(1))
			Expect(secrets.Items[0].Data["url"]).To(Equal([]byte("probably a git repo.git")))
			Expect(secrets.Items[0].Data["sshPrivateKey"]).To(Equal([]byte("some encrypted string")))
		})
	})
})
*/

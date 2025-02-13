/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"encoding/base64"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestHashData(t *testing.T) {
	testString1 := "My Wonderful Test String"
	testString2 := "Another Wonderful Test String"

	out1 := hashData(testString1)
	out2 := hashData(testString2)

	assert.Equal(
		t,
		// revive:disable-next-line
		"703fe1668c39ec0fdf3c9916d526ba4461fe10fd36bac1e2a1b708eb8a593e418eb3f92dbbd2a6e3776516b0e03743a45cfd69de6a3280afaa90f43fa1918f74",
		out1,
	)
	assert.Equal(
		t,
		// revive:disable-next-line
		"d3bfd910013886fe68ffd5c5d854e7cb2a8ce2a15a48ade41505b52ce7898f63d8e6b9c84eacdec33c45f7a2812d93732b524be91286de328bbd6b72d5aee9de",
		out2,
	)
}

var _ = Describe("Secret controller", func() {
	ctx := context.Background()

	var reconciler *PaasNSReconciler
	BeforeEach(func() {
		reconciler = &PaasNSReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	When("reconciling a PaasNS with no secrets", Ordered, func() {
		pns := &api.PaasNS{
			ObjectMeta: metav1.ObjectMeta{Name: "foo"},
			Spec: api.PaasNSSpec{
				Paas: "my-paas",
			},
		}

		It("should not return an error", func() {
			err := reconciler.ReconcileSecrets(ctx, &api.Paas{}, pns)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should not create any secrets", func() {
			secrets := &corev1.SecretList{}
			err := k8sClient.List(ctx, secrets, client.InNamespace("my-paas-foo"))

			Expect(err).NotTo(HaveOccurred())
			Expect(secrets.Items).To(BeZero())
		})
	})

	When("reconciling a PaasNS with an SshSecrets value", Ordered, func() {
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
					SshSecrets: map[string]string{
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
			err := reconciler.ReconcileSecrets(ctx, paas, pns)

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

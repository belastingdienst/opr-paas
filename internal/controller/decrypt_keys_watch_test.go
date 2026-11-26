package controller

import (
	"context"

	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("paasesForDecryptKeysSecret", func() {
	const (
		keysName      = "keys"
		keysNamespace = "paas-system"
	)
	ctx := context.Background()

	activeConfig := func() *v1alpha2.PaasConfig {
		return &v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "paas-config"},
			Spec: v1alpha2.PaasConfigSpec{
				DecryptKeysSecret: v1alpha2.NamespacedName{Name: keysName, Namespace: keysNamespace},
			},
			Status: v1alpha2.PaasConfigStatus{
				Conditions: []metav1.Condition{{Type: v1alpha2.TypeActivePaasConfig, Status: metav1.ConditionTrue}},
			},
		}
	}
	secret := func(name, namespace string) *corev1.Secret {
		return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	}
	newClient := func(objects ...client.Object) client.Client {
		return fake.NewClientBuilder().WithScheme(k8sClient.Scheme()).WithObjects(objects...).Build()
	}
	paas := func(name string) *v1alpha2.Paas {
		return &v1alpha2.Paas{ObjectMeta: metav1.ObjectMeta{Name: name}}
	}

	It("requeues every Paas when the decryption keys secret changes", func() {
		c := newClient(activeConfig(), paas("one"), paas("two"))
		Expect(paasesForDecryptKeysSecret(ctx, c, secret(keysName, keysNamespace))).To(ConsistOf(
			reconcile.Request{NamespacedName: types.NamespacedName{Name: "one"}},
			reconcile.Request{NamespacedName: types.NamespacedName{Name: "two"}},
		))
	})

	It("ignores other secrets", func() {
		c := newClient(activeConfig(), paas("one"))
		Expect(paasesForDecryptKeysSecret(ctx, c, secret("other", keysNamespace))).To(BeEmpty())
		Expect(paasesForDecryptKeysSecret(ctx, c, secret(keysName, "other-namespace"))).To(BeEmpty())
	})

	It("ignores secrets when there is no active PaasConfig", func() {
		inactive := activeConfig()
		inactive.Status.Conditions = nil
		c := newClient(inactive, paas("one"))
		Expect(paasesForDecryptKeysSecret(ctx, c, secret(keysName, keysNamespace))).To(BeEmpty())
	})
})

package controller

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/config"
	paasquota "github.com/belastingdienst/opr-paas/v3/pkg/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Namespace", Ordered, func() {
	const (
		paasRequestor = "ns-controller"
		manByPaas     = "man-paas"
		capName       = "argocd"
		ns1           = "ns1"
		ns2           = "ns2"
		lbl1Key       = "key1"
		lbl1Value     = "value1"
		lbl2Key       = "key2"
		lbl2Value     = "value2"
		manByLbl      = "manbylbl"
		manBySuffix   = "manby"
		reqLbl        = "requestor-label"
		qtaLbl        = "quota-label"
		kubeInstLabel = "app.kubernetes.io/instance"
	)
	var (
		ctx        context.Context
		paas       *v1alpha2.Paas
		reconciler *PaasReconciler
		myConfig   v1alpha2.PaasConfig
		paasName   = paasRequestor
	)

	BeforeEach(func() {
		ctx = context.Background()
		paasName = paasRequestor
		paas = &v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				UID:  "MY-UID",
				Name: paasName,
				Labels: map[string]string{
					lbl1Key:       lbl1Value,
					lbl2Key:       lbl2Value,
					kubeInstLabel: "whatever",
				},
			},
			Spec: v1alpha2.PaasSpec{
				ManagedByPaas: manByPaas,
				Requestor:     paasRequestor,
				Capabilities: v1alpha2.PaasCapabilities{
					capName: v1alpha2.PaasCapability{},
				},
				Quota: paasquota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
				Namespaces: v1alpha2.PaasNamespaces{
					ns1: {},
					ns2: {},
				},
			},
		}
		myConfig = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: v1alpha2.PaasConfigSpec{
				Capabilities: map[string]v1alpha2.ConfigCapability{
					capName: {
						QuotaSettings: v1alpha2.ConfigQuotaSettings{
							DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
								corev1.ResourceLimitsCPU: resourcev1.MustParse("5"),
							},
						},
					},
				},
				ManagedByLabel:  manByLbl,
				ManagedBySuffix: manBySuffix,
				RequestorLabel:  reqLbl,
				QuotaLabel:      qtaLbl,
				Templating: v1alpha2.ConfigTemplatingItems{
					NamespaceLabels: v1alpha2.ConfigTemplatingItem{
						//revive:disable-next-line
						"":       "{{ range $key, $value := .Paas.Labels }}{{ if ne $key \"" + kubeInstLabel + "\" }}{{$key}}: {{$value}}\n{{end}}{{end}}",
						manByLbl: "{{ .Paas.Spec.ManagedByPaas }}-" + manBySuffix,
					},
				},
			},
		}
		// Updates context to include paasConfig
		ctx = context.WithValue(ctx, config.ContextKeyPaasConfig, myConfig)

		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	When("reconciling namespaces for a paas", func() {
		var (
			nsDefs             namespaceDefs
			expectedNamespaces = []string{ns1, ns2, capName}
		)
		It("has proper nsDefs", func() {
			var err error
			nsDefs, err = reconciler.nsDefsFromPaas(ctx, paas)
			Expect(err).NotTo(HaveOccurred())
			Expect(nsDefs).To(HaveLen(len(expectedNamespaces)))
		})
		It("reconciles successfully", func() {
			err := reconciler.reconcileNamespaces(ctx, paas, nsDefs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates all namespaces as expected", func() {
			for _, nsName := range expectedNamespaces {
				var ns corev1.Namespace
				err := reconciler.Get(ctx, types.NamespacedName{Name: join(paasName, nsName)}, &ns)
				Expect(err).NotTo(HaveOccurred())
			}
		})
		It("have set all expected labels", func() {
			var (
				expectedLabels = map[string]string{
					lbl1Key:           lbl1Value,
					lbl2Key:           lbl2Value,
					ManagedByLabelKey: paasName,
					manByLbl:          join(manByPaas, manBySuffix),
				}
			)
			for nsName, nsDef := range nsDefs {
				fmt.Fprintf(GinkgoWriter, "DEBUG - Namespace: %v", nsName)
				var ns corev1.Namespace
				err := reconciler.Get(ctx, types.NamespacedName{Name: nsName}, &ns)
				Expect(err).NotTo(HaveOccurred())
				for key, value := range expectedLabels {
					Expect(ns.ObjectMeta.Labels).To(HaveKeyWithValue(key, value))
				}
				Expect(ns.ObjectMeta.Labels).To(HaveKeyWithValue(qtaLbl, nsDef.quotaName))
				Expect(ns.ObjectMeta.Labels).NotTo(HaveKey(kubeInstLabel))
			}
		})
	})
})

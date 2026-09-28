package controller

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v5/internal/config"
	"github.com/belastingdienst/opr-paas/v5/internal/utils"
	paasquota "github.com/belastingdienst/opr-paas/v5/pkg/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Cluster Quotas", Ordered, func() {
	const (
		paasRequestor = "cq-controller"
		capName       = "argocd"
		lbl1Key       = "key1"
		lbl1Value     = "value1"
		lbl2Key       = "key2"
		lbl2Value     = "value2"
		manByLbl      = "manbylbl"
		manBySuffix   = "manby"
		reqLbl        = "requestor-label"
		qtaLbl        = "quota-label"
		kubeInstLabel = "app.kubernetes.io/instance"
		nsName        = "testNamespace"
	)
	var (
		paas           *v1alpha2.Paas
		paasNoQuota    *v1alpha2.Paas
		paasEmptyQuota *v1alpha2.Paas
		reconciler     *PaasReconciler
		myConfig       v1alpha2.PaasConfig
		paasName       = paasRequestor
	)
	ctx := context.Background()

	BeforeEach(func() {
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
				Requestor: paasRequestor,
				Namespaces: v1alpha2.PaasNamespaces{
					nsName: v1alpha2.PaasNamespace{},
				},
				Capabilities: v1alpha2.PaasCapabilities{
					capName: v1alpha2.PaasCapability{},
				},
				Quota: paasquota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
			},
		}
		paasNoQuota = &v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				UID:  "MY-UID",
				Name: "paas-no-quota",
				Labels: map[string]string{
					lbl1Key:       lbl1Value,
					lbl2Key:       lbl2Value,
					kubeInstLabel: "whatever",
				},
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: paasRequestor,
				Namespaces: v1alpha2.PaasNamespaces{
					nsName: v1alpha2.PaasNamespace{},
				},
				Capabilities: v1alpha2.PaasCapabilities{
					capName: v1alpha2.PaasCapability{},
				},
			},
		}
		paasEmptyQuota = &v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				UID:  "MY-UID",
				Name: "paas-empty-quota",
				Labels: map[string]string{
					lbl1Key:       lbl1Value,
					lbl2Key:       lbl2Value,
					kubeInstLabel: "whatever",
				},
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: paasRequestor,
				Namespaces: v1alpha2.PaasNamespaces{
					nsName: v1alpha2.PaasNamespace{},
				},
				Capabilities: v1alpha2.PaasCapabilities{
					capName: v1alpha2.PaasCapability{},
				},
				Quota: paasquota.Quota{},
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
				FeatureFlags: v1alpha2.ConfigFeatureFlags{
					ClusterResourceQuotaManagement: "allow",
				},
				Templating: v1alpha2.ConfigTemplatingItems{
					ClusterQuotaLabels: v1alpha2.ConfigTemplatingItem{
						//revive:disable-next-line
						"": "{{ range $key, $value := .Paas.Labels }}{{ if ne $key \"" + kubeInstLabel + "\" }}" +
							"{{$key}}: {{$value}}\n{{end}}{{end}}",
					},
				},
			},
		}

		// Updates context to include paasConfig
		ctx = context.WithValue(context.Background(), config.ContextKeyPaasConfig, myConfig)

		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	// getPaasFromRequest
	When("reconciling quotas for a paas", func() {
		expectedQuotas := []string{paasName, utils.Join(paasName, capName)}

		It("reconciles successfully", func() {
			err := reconciler.reconcileQuotas(ctx, paas)
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates all cluster resource quotas as expected", func() {
			for _, quotaName := range expectedQuotas {
				var quota quotav1.ClusterResourceQuota
				err := reconciler.Get(ctx, types.NamespacedName{Name: quotaName}, &quota)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("have set all expected labels", func() {
			expectedLabels := map[string]string{
				lbl1Key: lbl1Value,
				lbl2Key: lbl2Value,
			}
			for _, quotaName := range expectedQuotas {
				fmt.Fprintf(GinkgoWriter, "DEBUG - Quota: %v\n", quotaName)
				var quota quotav1.ClusterResourceQuota
				err := reconciler.Get(ctx, types.NamespacedName{Name: quotaName}, &quota)
				Expect(err).NotTo(HaveOccurred())
				for key, value := range expectedLabels {
					Expect(quota.ObjectMeta.Labels).To(HaveKeyWithValue(key, value))
				}
				Expect(quota.ObjectMeta.Labels).NotTo(HaveKey(kubeInstLabel))
			}
		})
	})

	When("quota management is not enabled", func() {
		for _, setting := range []string{"warn", "block"} {
			disabledPaasName := "paas-disabled-" + setting
			expectedQuotas := []string{disabledPaasName, utils.Join(disabledPaasName, capName)}

			buildDisabledPaas := func() *v1alpha2.Paas {
				return &v1alpha2.Paas{
					ObjectMeta: metav1.ObjectMeta{
						UID:  types.UID("MY-UID-" + setting),
						Name: disabledPaasName,
					},
					Spec: v1alpha2.PaasSpec{
						Requestor: paasRequestor,
						Namespaces: v1alpha2.PaasNamespaces{
							nsName: v1alpha2.PaasNamespace{},
						},
						Capabilities: v1alpha2.PaasCapabilities{
							capName: v1alpha2.PaasCapability{},
						},
						Quota: paasquota.Quota{
							"cpu": resourcev1.MustParse("1"),
						},
					},
				}
			}

			It(fmt.Sprintf("does not create any cluster resource quotas (setting: %s)", setting), func() {
				myConfig.Spec.FeatureFlags.ClusterResourceQuotaManagement = setting
				ctx = context.WithValue(context.Background(), config.ContextKeyPaasConfig, myConfig)

				err := reconciler.reconcileQuotas(ctx, buildDisabledPaas())
				Expect(err).NotTo(HaveOccurred())

				for _, quotaName := range expectedQuotas {
					var quota quotav1.ClusterResourceQuota
					err = reconciler.Get(ctx, types.NamespacedName{Name: quotaName}, &quota)
					Expect(err).To(HaveOccurred())
				}
			})

			It(fmt.Sprintf("deletes existing cluster resource quotas (setting: %s)", setting), func() {
				disabledPaas := buildDisabledPaas()

				// First reconcile while enabled, so the quotas get created.
				Expect(reconciler.reconcileQuotas(ctx, disabledPaas)).NotTo(HaveOccurred())

				// Check that expected quotas exist
				for _, quotaName := range expectedQuotas {
					var quota quotav1.ClusterResourceQuota
					err := reconciler.Get(ctx, types.NamespacedName{Name: quotaName}, &quota)
					Expect(err).NotTo(HaveOccurred())
				}

				// Disable quota management and reconcile again.
				myConfig.Spec.FeatureFlags.ClusterResourceQuotaManagement = setting
				ctx = context.WithValue(context.Background(), config.ContextKeyPaasConfig, myConfig)
				Expect(reconciler.reconcileQuotas(ctx, disabledPaas)).NotTo(HaveOccurred())

				// Check that existing quotas were deleted
				for _, quotaName := range expectedQuotas {
					var quota quotav1.ClusterResourceQuota
					err := reconciler.Get(ctx, types.NamespacedName{Name: quotaName}, &quota)
					Expect(k8serrors.IsNotFound(err)).To(BeTrue())
				}
			})
		}
	})

	When("reconciling quota's for a paas with an empty quota block", func() {
		It("reconciles successfully", func() {
			err := reconciler.reconcileQuotas(ctx, paasEmptyQuota)
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates all cluster resource quotas as expected", func() {
			var quota quotav1.ClusterResourceQuota
			err := reconciler.Get(ctx, types.NamespacedName{Name: "paas-empty-quota"}, &quota)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("clusterresourcequotas.quota.openshift.io" +
				" \"paas-empty-quota\" not found"))
		})
	})

	When("reconciling quota's for a paas without a quota block", func() {
		It("reconciles successfully", func() {
			err := reconciler.reconcileQuotas(ctx, paasNoQuota)
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates all cluster resource quotas as expected", func() {
			var quota quotav1.ClusterResourceQuota
			err := reconciler.Get(ctx, types.NamespacedName{Name: "paas-no-quota"}, &quota)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("clusterresourcequotas.quota.openshift.io" +
				" \"paas-no-quota\" not found"))
		})
	})
})

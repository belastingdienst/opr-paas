package controller

import (
	"context"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ClusterResourceQuota controller", Ordered, func() {
	const (
		paasName = "crq-test-paas"
		capName  = "crq-test-cap"
	)

	var (
		ctx        context.Context
		reconciler *PaasReconciler
		paasConfig v1alpha2.PaasConfig
		paas       *api.Paas
	)
	var quotaName = types.NamespacedName{Name: "paas-" + capName}

	BeforeEach(func() {
		ctx = context.Background()
		paasConfig = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: v1alpha2.PaasConfigSpec{
				Capabilities: map[string]v1alpha2.ConfigCapability{
					capName: {
						QuotaSettings: v1alpha2.ConfigQuotaSettings{
							Clusterwide: true,
							Ratio:       1,
							DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
								corev1.ResourceLimitsCPU:   resourcev1.MustParse("3"),
								corev1.ResourceRequestsCPU: resourcev1.MustParse("1"),
							},
						},
					},
				},
			},
		}
		config.SetConfig(paasConfig)

		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		paas = &api.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
			},
			Spec: api.PaasSpec{
				Requestor: "foo",
				Quota:     quota.Quota{},
				Capabilities: api.PaasCapabilities{
					capName: api.PaasCapability{
						Enabled: true,
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, paas)).NotTo(HaveOccurred())

		Expect(reconciler.addToClusterWideQuota(ctx, paas, capName)).
			NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, paas)).NotTo(HaveOccurred())
	})

	When("a capability is configured with a cluster-wide quota", func() {
		It("should create a ClusterResourceQuota with the configured defaults "+
			"when reconciling a Paas with the capability", func() {
			q := &quotav1.ClusterResourceQuota{}
			Expect(k8sClient.Get(ctx, quotaName, q)).
				NotTo(HaveOccurred())
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceLimitsCPU, resourcev1.DecimalSI).String()).
				To(Equal("3"))
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceRequestsCPU, resourcev1.DecimalSI).String()).
				To(Equal("1"))
		})

		It("should update the ClusterResourceQuota when changing the capability quota", func() {
			paas.Spec.Capabilities[capName] = api.PaasCapability{
				Enabled: true,
				Quota: quota.Quota{
					corev1.ResourceLimitsCPU:   resourcev1.MustParse("1500m"),
					corev1.ResourceRequestsCPU: resourcev1.MustParse("500m"),
				},
			}
			Expect(k8sClient.Update(ctx, paas)).
				NotTo(HaveOccurred())
			Expect(reconciler.addToClusterWideQuota(ctx, paas, capName)).
				NotTo(HaveOccurred())

			q := &quotav1.ClusterResourceQuota{}
			Expect(k8sClient.Get(ctx, quotaName, q)).
				NotTo(HaveOccurred())
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceLimitsCPU, resourcev1.DecimalSI).String()).
				To(Equal("1500m"))
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceRequestsCPU, resourcev1.DecimalSI).String()).
				To(Equal("500m"))
		})

		It("should remove the ClusterResourceQuota when the capability is disabled", func() {
			// The k8s client strips type meta, but it's needed for equality checks within our reconciler
			paas.TypeMeta = metav1.TypeMeta{
				APIVersion: api.GroupVersion.String(),
				Kind:       "Paas",
			}
			paas.Spec.Capabilities[capName] = api.PaasCapability{
				Enabled: false,
			}
			Expect(reconciler.reconcileClusterWideQuota(ctx, paas)).
				NotTo(HaveOccurred())

			q := &quotav1.ClusterResourceQuota{}
			err := k8sClient.Get(ctx, quotaName, q)
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should remove the ClusterResourceQuota on finalization", func() {
			paas.TypeMeta = metav1.TypeMeta{
				APIVersion: api.GroupVersion.String(),
				Kind:       "Paas",
			}
			Expect(reconciler.finalizeClusterWideQuotas(ctx, paas)).
				NotTo(HaveOccurred())

			q := &quotav1.ClusterResourceQuota{}
			err := k8sClient.Get(ctx, quotaName, q)
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should increase quota of the ClusterResourceQuota for additional Paas' with the capability", func() {
			paas2 := &api.Paas{
				ObjectMeta: metav1.ObjectMeta{
					Name: paasName + "-2",
				},
				Spec: api.PaasSpec{
					Requestor: "bar",
					Quota:     quota.Quota{},
					Capabilities: api.PaasCapabilities{
						capName: api.PaasCapability{
							Enabled: true,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, paas2)).NotTo(HaveOccurred())

			Expect(reconciler.addToClusterWideQuota(ctx, paas2, capName)).
				NotTo(HaveOccurred())

			q := &quotav1.ClusterResourceQuota{}
			Expect(k8sClient.Get(ctx, quotaName, q)).
				NotTo(HaveOccurred())
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceLimitsCPU, resourcev1.DecimalSI).String()).
				To(Equal("6"))
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceRequestsCPU, resourcev1.DecimalSI).String()).
				To(Equal("2"))

			Expect(k8sClient.Delete(ctx, paas2)).NotTo(HaveOccurred())
		})

		It("should decrease quota of the ClusterResourceQuota when a Paas removes the capability", func() {
			paas2 := &api.Paas{
				ObjectMeta: metav1.ObjectMeta{
					Name: paasName + "-2",
				},
				Spec: api.PaasSpec{
					Requestor: "bar",
					Quota:     quota.Quota{},
					Capabilities: api.PaasCapabilities{
						capName: api.PaasCapability{
							Enabled: true,
							Quota: quota.Quota{
								corev1.ResourceLimitsCPU:   resourcev1.MustParse("1500m"),
								corev1.ResourceRequestsCPU: resourcev1.MustParse("500m"),
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, paas2)).NotTo(HaveOccurred())

			Expect(reconciler.addToClusterWideQuota(ctx, paas2, capName)).
				NotTo(HaveOccurred())
			paas.TypeMeta = metav1.TypeMeta{
				APIVersion: api.GroupVersion.String(),
				Kind:       "Paas",
			}
			Expect(reconciler.removeFromClusterWideQuota(ctx, paas, capName)).
				NotTo(HaveOccurred())

			q := &quotav1.ClusterResourceQuota{}
			Expect(k8sClient.Get(ctx, quotaName, q)).
				NotTo(HaveOccurred())
			Expect(q.OwnerReferences).To(HaveLen(1))
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceLimitsCPU, resourcev1.DecimalSI).String()).
				To(Equal("1500m"))
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceRequestsCPU, resourcev1.DecimalSI).String()).
				To(Equal("500m"))
		})
	})
})

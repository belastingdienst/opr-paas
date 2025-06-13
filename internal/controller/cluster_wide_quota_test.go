package controller

import (
	"context"
	"strconv"

	"github.com/belastingdienst/opr-paas/v2/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v2/internal/config"
	"github.com/belastingdienst/opr-paas/v2/internal/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ClusterResourceQuota controller", func() {
	const (
		capName    = "crq-test-cap"
		paasPrefix = "crq-test-paas"
	)

	var (
		ctx        context.Context
		reconciler *PaasReconciler
		paasConfig v1alpha2.PaasConfig
		paas       *v1alpha2.Paas
		quotaName  = types.NamespacedName{Name: join("paas", capName)}
	)

	addPaasWithCap := func(name string, pc v1alpha2.PaasCapability) *v1alpha2.Paas {
		paas = &v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: "foo",
				Quota:     quota.Quota{},
				Capabilities: v1alpha2.PaasCapabilities{
					capName: pc,
				},
			},
		}
		Expect(k8sClient.Create(ctx, paas)).NotTo(HaveOccurred())
		// The k8s client strips type meta, but it's needed for equality checks within our reconciler
		paas.TypeMeta = metav1.TypeMeta{
			APIVersion: v1alpha2.GroupVersion.String(),
			Kind:       "Paas",
		}

		Expect(reconciler.addToClusterWideQuota(ctx, paas, capName)).NotTo(HaveOccurred())
		return paas
	}
	addPaasWithDefCap := func(name string) *v1alpha2.Paas {
		return addPaasWithCap(name, v1alpha2.PaasCapability{})
	}

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
	})

	AfterEach(func() {
		q := &quotav1.ClusterResourceQuota{}
		if k8sClient.Get(ctx, quotaName, q) == nil {
			Expect(k8sClient.Delete(ctx, q)).NotTo(HaveOccurred())
		}

		ps := &v1alpha2.PaasList{}
		Expect(k8sClient.List(ctx, ps)).NotTo(HaveOccurred())
		for _, p := range ps.Items {
			Expect(k8sClient.Delete(ctx, &p)).NotTo(HaveOccurred())
		}
	})

	When("a capability is configured with a cluster-wide quota", func() {
		It("should create a ClusterResourceQuota with the configured defaults "+
			"when reconciling a Paas with the capability", func() {
			addPaasWithDefCap(paasPrefix)
			q := &quotav1.ClusterResourceQuota{}
			Expect(k8sClient.Get(ctx, quotaName, q)).
				NotTo(HaveOccurred())
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceLimitsCPU, resourcev1.DecimalSI).String()).
				To(Equal("3"))
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceRequestsCPU, resourcev1.DecimalSI).String()).
				To(Equal("1"))
		})

		It("should update the ClusterResourceQuota when changing the capability quota", func() {
			paas := addPaasWithDefCap(paasPrefix)
			paas.Spec.Capabilities[capName] = v1alpha2.PaasCapability{
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
			paas := addPaasWithDefCap(paasPrefix)
			delete(paas.Spec.Capabilities, capName)
			Expect(reconciler.reconcileClusterWideQuota(ctx, paas)).
				NotTo(HaveOccurred())

			q := &quotav1.ClusterResourceQuota{}
			err := k8sClient.Get(ctx, quotaName, q)
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should remove the ClusterResourceQuota on finalization", func() {
			paas := addPaasWithDefCap(paasPrefix)
			Expect(reconciler.finalizeClusterWideQuotas(ctx, paas)).
				NotTo(HaveOccurred())

			q := &quotav1.ClusterResourceQuota{}
			err := k8sClient.Get(ctx, quotaName, q)
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should increase quota of the ClusterResourceQuota for additional Paas' with the capability", func() {
			addPaasWithDefCap(paasPrefix)
			addPaasWithDefCap(join(paasPrefix, "2"))

			q := &quotav1.ClusterResourceQuota{}
			Expect(k8sClient.Get(ctx, quotaName, q)).
				NotTo(HaveOccurred())
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceLimitsCPU, resourcev1.DecimalSI).String()).
				To(Equal("6"))
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceRequestsCPU, resourcev1.DecimalSI).String()).
				To(Equal("2"))
		})

		It("should decrease quota of the ClusterResourceQuota when a Paas removes the capability", func() {
			paas1 := addPaasWithDefCap(join(paasPrefix, "1"))
			addPaasWithCap(
				join(paasPrefix, "2"),
				v1alpha2.PaasCapability{
					Quota: quota.Quota{
						corev1.ResourceLimitsCPU:   resourcev1.MustParse("1500m"),
						corev1.ResourceRequestsCPU: resourcev1.MustParse("500m"),
					},
				},
			)

			Expect(reconciler.removeFromClusterWideQuota(ctx, paas1, capName)).
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

	When("a capability is configured with a cluster-wide quota minimum and maximum", func() {
		BeforeEach(func() {
			conf := config.GetConfig()
			c := conf.Spec.Capabilities[capName]
			c.QuotaSettings.MinQuotas = map[corev1.ResourceName]resourcev1.Quantity{
				corev1.ResourceLimitsCPU:   resourcev1.MustParse("9"),
				corev1.ResourceRequestsCPU: resourcev1.MustParse("3"),
			}
			c.QuotaSettings.MaxQuotas = map[corev1.ResourceName]resourcev1.Quantity{
				corev1.ResourceLimitsCPU:   resourcev1.MustParse("18"),
				corev1.ResourceRequestsCPU: resourcev1.MustParse("6"),
			}
			conf.Spec.Capabilities[capName] = c
			config.SetConfig(conf)
		})

		It("should create a ClusterResourceQuota with the minimum quota when reconciling Paas' "+
			"whose sum of capability quotas is less than the minimum", func() {
			addPaasWithDefCap(join(paasPrefix, "1"))
			addPaasWithDefCap(join(paasPrefix, "2"))

			q := &quotav1.ClusterResourceQuota{}
			Expect(k8sClient.Get(ctx, quotaName, q)).
				NotTo(HaveOccurred())
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceLimitsCPU, resourcev1.DecimalSI).String()).
				To(Equal("9"))
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceRequestsCPU, resourcev1.DecimalSI).String()).
				To(Equal("3"))
		})

		It("should create a ClusterResourceQuota with the sum of Paas quotas "+
			"when the sum is between the minimum and maximum", func() {
			for i := 1; i < 5; i++ {
				addPaasWithDefCap(join(paasPrefix, strconv.Itoa(i)))
			}
			addPaasWithCap(
				join(paasPrefix, "5"),
				v1alpha2.PaasCapability{
					Quota: quota.Quota{
						corev1.ResourceLimitsCPU: resourcev1.MustParse("5"),
					},
				},
			)

			q := &quotav1.ClusterResourceQuota{}
			Expect(k8sClient.Get(ctx, quotaName, q)).
				NotTo(HaveOccurred())
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceLimitsCPU, resourcev1.DecimalSI).String()).
				To(Equal("17"))
		})

		It("should create a ClusterResourceQuota with the maximum when the sum of Paas' quotas exceeds it", func() {
			addPaasWithDefCap(join(paasPrefix, "1"))
			addPaasWithCap(
				join(paasPrefix, "2"),
				v1alpha2.PaasCapability{
					Quota: quota.Quota{
						corev1.ResourceLimitsCPU: resourcev1.MustParse("17"),
					},
				},
			)

			q := &quotav1.ClusterResourceQuota{}
			Expect(k8sClient.Get(ctx, quotaName, q)).
				NotTo(HaveOccurred())
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceLimitsCPU, resourcev1.DecimalSI).String()).
				To(Equal("18"))
		})

		It("should create a ClusterResourceQuota with the sum of Paas' quotas multiplied by the ratio "+
			"when the capability is configured with a ratio other than 1", func() {
			conf := config.GetConfig()
			c := conf.Spec.Capabilities[capName]
			c.QuotaSettings.Ratio = 1.4
			conf.Spec.Capabilities[capName] = c
			config.SetConfig(conf)

			addPaasWithDefCap(join(paasPrefix, "1"))
			addPaasWithCap(
				join(paasPrefix, "2"),
				v1alpha2.PaasCapability{
					Quota: quota.Quota{
						corev1.ResourceLimitsCPU: resourcev1.MustParse("8"),
					},
				},
			)

			q := &quotav1.ClusterResourceQuota{}
			Expect(k8sClient.Get(ctx, quotaName, q)).
				NotTo(HaveOccurred())
			Expect(q.Spec.Quota.Hard.Name(corev1.ResourceLimitsCPU, resourcev1.DecimalSI).String()).
				To(Equal("15"))
		})
	})
})

/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/fields"
	paasquota "github.com/belastingdienst/opr-paas/internal/quota"
	argocd "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func patchAppSet(ctx context.Context, newAppSet *argocd.ApplicationSet) {
	oldAppSet := &argocd.ApplicationSet{}
	namespacedName := types.NamespacedName{
		Name:      newAppSet.Name,
		Namespace: newAppSet.Namespace,
	}
	err := k8sClient.Get(ctx, namespacedName, oldAppSet)
	if err == nil {
		// Patch
		patch := client.MergeFrom(oldAppSet.DeepCopy())
		oldAppSet.Spec = newAppSet.Spec
		err = k8sClient.Patch(ctx, oldAppSet, patch)
		Expect(err).NotTo(HaveOccurred())
	} else {
		Expect(err.Error()).To(MatchRegexp(`applicationsets.argoproj.io .* not found`))
		err = k8sClient.Create(ctx, newAppSet)
		Expect(err).NotTo(HaveOccurred())
	}
}

func assureNamespace(ctx context.Context, namespaceName string) {
	oldNs := &corev1.Namespace{}
	namespacedName := types.NamespacedName{
		Name: namespaceName,
	}
	err := k8sClient.Get(ctx, namespacedName, oldNs)
	if err == nil {
		return
	}
	Expect(err.Error()).To(MatchRegexp(`namespaces .* not found`))
	err = k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
	})
	Expect(err).NotTo(HaveOccurred())
}

func assurePaas(ctx context.Context, newPaas *api.Paas) {
	oldPaas := &api.Paas{}
	namespacedName := types.NamespacedName{
		Name: newPaas.Name,
	}
	err := k8sClient.Get(ctx, namespacedName, oldPaas)
	if err == nil {
		return
	}
	Expect(err.Error()).To(MatchRegexp(`paas.cpet.belastingdienst.nl .* not found`))
	err = k8sClient.Create(ctx, newPaas)
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("Paas Controller", Ordered, func() {
	const (
		paasRequestor      = "paas-controller"
		capAppSetNamespace = "asns"
		capAppSetName      = "argoas"
		capName            = "argocd"
	)
	var (
		paas         *api.Paas
		appSet       *argocd.ApplicationSet
		reconciler   *PaasReconciler
		request      controllerruntime.Request
		myConfig     v1alpha2.PaasConfig
		paasName     = paasRequestor
		capNamespace = paasName + "-" + capName
	)
	ctx := context.Background()

	BeforeAll(func() {
		assureNamespace(ctx, "gsns")
		appSet = &argocd.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      capAppSetName,
				Namespace: capAppSetNamespace,
			},
			Spec: argocd.ApplicationSetSpec{
				Generators: []argocd.ApplicationSetGenerator{},
			},
		}
	})

	BeforeEach(func() {
		paas = &api.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
			},
			Spec: api.PaasSpec{
				Requestor: paasRequestor,
				Capabilities: api.PaasCapabilities{
					capName: api.PaasCapability{
						Enabled: true,
					},
				},
				Quota: paasquota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
			},
		}
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
				Debug:           false,
				ManagedByLabel:  "argocd.argoproj.io/manby",
				ManagedBySuffix: "argocd",
				RequestorLabel:  "o.lbl",
				QuotaLabel:      "q.lbl",
				GroupSyncList: v1alpha2.NamespacedName{
					Namespace: "gsns",
					Name:      "wlname",
				},
				GroupSyncListKey: "groupsynclist.txt",
			},
		}
		config.SetConfig(myConfig)
		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	When("reconciling a Paas with argocd capability", func() {
		It("should not return an error", func() {
			paasName = paasRequestor + "-normal"
			paas.Name = paasName
			request.Name = paasName
			capNamespace = paasName + "-" + capName
			assurePaas(ctx, paas)
			assureNamespace(ctx, capNamespace)
			patchAppSet(ctx, appSet)
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter.Microseconds()).To(BeZero())
		})
		It("should create an appset entry", func() {
			appSet := &argocd.ApplicationSet{}
			appSetName := types.NamespacedName{
				Name:      capAppSetName,
				Namespace: capAppSetNamespace,
			}
			err := k8sClient.Get(ctx, appSetName, appSet)
			Expect(err).NotTo(HaveOccurred())
			entries := make(fields.Entries)
			for _, generator := range appSet.Spec.Generators {
				generatorEntries, err := fields.EntriesFromJSON(generator.List.Elements)
				Expect(err).NotTo(HaveOccurred())
				entries = entries.Merge(generatorEntries)
			}
			Expect(entries).To(HaveKey(paasName))
		})
	})

	When("reconciling a Paas without argocd capability", func() {
		It("should not return an error", func() {
			paasName = paasRequestor + "-nocap"
			paas.Name = paasName
			paas.Spec.Capabilities = make(api.PaasCapabilities)
			request.Name = paasName
			capNamespace = paasName + "-" + capName
			assurePaas(ctx, paas)
			assureNamespace(ctx, capNamespace)
			patchAppSet(ctx, appSet)
			paas.Spec.Capabilities = make(api.PaasCapabilities)
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter.Microseconds()).To(BeZero())
		})

		It("should not create an appset entry", func() {
			appSet := &argocd.ApplicationSet{}
			appSetName := types.NamespacedName{
				Name:      capAppSetName,
				Namespace: capAppSetNamespace,
			}
			err := k8sClient.Get(ctx, appSetName, appSet)
			Expect(err).NotTo(HaveOccurred())
			entries := make(fields.Entries)
			for _, generator := range appSet.Spec.Generators {
				generatorEntries, err := fields.EntriesFromJSON(generator.List.Elements)
				Expect(err).NotTo(HaveOccurred())
				entries = entries.Merge(generatorEntries)
			}
			Expect(entries).NotTo(HaveKey(paasName))
		})
	})
})

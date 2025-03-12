/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	paasquota "github.com/belastingdienst/opr-paas/internal/quota"
	appv1 "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Paas Controller", Ordered, func() {
	const (
		paasName           = "paas-controller-paas"
		paasRequestor      = "paas-controller"
		capAppSetNamespace = "asns"
		capAppSetName      = "argoas"
	)
	var (
		paas       *api.Paas
		reconciler *PaasReconciler
		request    controllerruntime.Request
		myConfig   api.PaasConfig
	)
	ctx := context.Background()

	BeforeAll(func() {
		paas = &api.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
			},
			Spec: api.PaasSpec{
				Requestor: paasRequestor,
				Capabilities: api.PaasCapabilities{
					"argocd": api.PaasCapability{
						Enabled: true,
					},
				},
				Quota: paasquota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
			},
		}
		err := k8sClient.Create(ctx, paas)
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "gsns"},
		})
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: paasName + "-argocd"},
		})
		Expect(err).NotTo(HaveOccurred())
		appSet := &appv1.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      capAppSetName,
				Namespace: capAppSetNamespace,
			},
			Spec: appv1.ApplicationSetSpec{
				Generators: []appv1.ApplicationSetGenerator{},
				Template: appv1.ApplicationSetTemplate{
					ApplicationSetTemplateMeta: appv1.ApplicationSetTemplateMeta{},
					Spec: appv1.ApplicationSpec{
						Destination: appv1.ApplicationDestination{
							Server:    "somewhere.org",
							Namespace: "default",
							Name:      "somewhere",
						},
						Project: "someproj",
					},
				},
			},
		}
		err = k8sClient.Create(ctx, appSet)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		myConfig = api.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: api.PaasConfigSpec{
				ClusterWideArgoCDNamespace: "asns",
				ArgoPermissions: api.ConfigArgoPermissions{
					ResourceName:  "argocd",
					DefaultPolicy: "role:tester",
					Role:          "admin",
					Header:        "g, system:cluster-admins, role:admin",
				},
				Capabilities: map[string]api.ConfigCapability{
					"argocd": {
						AppSet: "argoas",
						QuotaSettings: api.ConfigQuotaSettings{
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
				GroupSyncList: api.NamespacedName{
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
		request.Name = paasName
	})

	When("reconciling a Paas", func() {
		It("should not return an error", func() {
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter.Microseconds()).To(BeZero())
		})
	})
})

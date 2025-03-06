/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	appv1 "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Capabilities controller", Ordered, func() {
	var (
		ctx        context.Context
		paas       *api.Paas
		reconciler *PaasReconciler
		appSet     *appv1.ApplicationSet
		capName    string
		paasConfig api.PaasConfig
	)

	BeforeAll(func() {
		paas = &api.Paas{ObjectMeta: metav1.ObjectMeta{
			Name: "my-paas",
			UID:  "abc", // Needed or owner references fail
		}}
		capName = "argocd"
	})

	BeforeEach(func() {
		ctx = context.Background()
		paasConfig = api.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: api.PaasConfigSpec{
				ClusterWideArgoCDNamespace: "asns",
				Capabilities: map[string]api.ConfigCapability{
					"argocd": {
						AppSet: "argoas",
					},
				},
			},
		}
		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		appSet = &appv1.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "argoas",
				Namespace: "asns",
			},
			Spec: appv1.ApplicationSetSpec{
				Generators: []appv1.ApplicationSetGenerator{},
				Template: appv1.ApplicationSetTemplate{
					ApplicationSetTemplateMeta: appv1.ApplicationSetTemplateMeta{},
					Spec: appv1.ApplicationSpec{
						Destination: appv1.ApplicationDestination{},
						Project:     "",
					},
				},
			},
		}
		config.SetConfig(paasConfig)
		err := k8sClient.Create(ctx, appSet)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := k8sClient.Delete(ctx, appSet)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when all is good", func() {
		It("ensuring capability in the AppSet should succeed", func() {
			err := reconciler.ensureAppSetCap(ctx, paas, capName)
			Expect(err).NotTo(HaveOccurred())
		})
		It("should error when appSet doesn't exist", func() {
			argoCapConfig := paasConfig.Spec.Capabilities["argocd"]
			argoCapConfig.AppSet = "doesnotexist"
			paasConfig.Spec.Capabilities["argocd"] = argoCapConfig
			config.SetConfig(paasConfig)
			// Delete appSet
			err := reconciler.ensureAppSetCap(ctx, paas, capName)
			Expect(err).To(HaveOccurred())
		})
	})
})

/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"context"

	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v4/pkg/fields"
	"github.com/belastingdienst/opr-paas/v4/pkg/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	paasWithArgo        = "paas-capability-argocd"
	paasArgoGitURL      = "ssh://git@scm/some-repo.git"
	paasArgoGitPath     = "foo/"
	paasArgoGitRevision = "main"
	paasRequestor       = "paas-requestor"
)

var _ = Describe("Service", func() {
	var (
		svc  *Service
		conf v1alpha2.PaasConfig
		ctx  context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		conf = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "example-paasconfig",
			},
			Spec: v1alpha2.PaasConfigSpec{
				Capabilities: map[string]v1alpha2.ConfigCapability{
					"argocd": {
						CustomFields: map[string]v1alpha2.ConfigCustomField{
							"git_url": {
								Required: true,
								// in yaml you need escaped slashes: '^ssh:\/\/git@scm\/[a-zA-Z0-9-.\/]*.git$'
								Validation: "^ssh://git@scm/[a-zA-Z0-9-./]*.git$",
							},
							"git_revision": {
								Default: "main",
							},
							"git_path": {
								Default: ".",
								// in yaml you need escaped slashes: '^[a-zA-Z0-9.\/]*$'
								Validation: "^[a-zA-Z0-9./]*$",
							},
						},
						QuotaSettings: v1alpha2.ConfigQuotaSettings{
							DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
								"argocd": resourcev1.MustParse("1"),
							},
						},
					},
				},
				DecryptKeysSecret: v1alpha2.NamespacedName{
					Name:      "name",
					Namespace: "namespace",
				},
				Templating: v1alpha2.ConfigTemplatingItems{
					GenericCapabilityFields: v1alpha2.ConfigTemplatingItem{
						"requestor":  "{{ .Paas.Spec.Requestor }}",
						"Service":    "{{ (split \"-\" .Paas.Name)._0 }}",
						"subservice": "{{ (split \"-\" .Paas.Name)._1 }}",
					},
				},
			},
		}
		err := k8sClient.Create(ctx, &conf)
		Expect(err).To(Not(HaveOccurred()))

		latest := &v1alpha2.PaasConfig{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latest)
		Expect(err).NotTo(HaveOccurred())

		meta.SetStatusCondition(&latest.Status.Conditions, metav1.Condition{
			Type:   v1alpha2.TypeActivePaasConfig,
			Status: metav1.ConditionTrue, Reason: "Reconciling", ObservedGeneration: latest.Generation,
			Message: "This config is the active config!",
		})
		err = k8sClient.Status().Update(ctx, latest)
		Expect(err).NotTo(HaveOccurred())

		svc = NewService(k8sClient)
	})

	AfterEach(func() {
		latest := &v1alpha2.PaasConfig{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: conf.Name}, latest)
		if !apierrors.IsNotFound(err) {
			err = k8sClient.Delete(ctx, &conf)
			Expect(err).To(Not(HaveOccurred()))
		}
	})

	Context("Generate", func() {
		It("returns templated capability elements from Paas CRs", func() {
			By("Creating a Paas with a capability")

			paas := &v1alpha2.Paas{
				ObjectMeta: metav1.ObjectMeta{
					Name: paasWithArgo,
				},
				Spec: v1alpha2.PaasSpec{
					Requestor: paasRequestor,
					Quota:     quota.Quota{},
					Capabilities: map[string]v1alpha2.PaasCapability{
						"argocd": {
							CustomFields: map[string]string{
								"git_url":      paasArgoGitURL,
								"git_path":     paasArgoGitPath,
								"git_revision": paasArgoGitRevision,
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, paas)).To(Succeed())

			By("Calling Generate")

			params := fields.ElementMap{
				"capability": "argocd",
			}
			results, err := svc.Generate(ctx, params)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).NotTo(BeEmpty())

			Expect(results[0]).To(Equal(fields.ElementMap{
				"git_path":     paasArgoGitPath,
				"git_revision": paasArgoGitRevision,
				"git_url":      paasArgoGitURL,
				"paas":         paasWithArgo,
				"requestor":    paasRequestor,
				"Service":      "paas",
				"subservice":   "capability",
			}))

			By("Calling Generate with a non-existent capability")

			params = fields.ElementMap{
				"capability": "nonexistent",
			}
			results, err = svc.Generate(ctx, params)
			Expect(err).To(HaveOccurred())
			Expect(results).To(BeEmpty())

			By("Calling Generate with no param")

			params = fields.ElementMap{}
			_, err = svc.Generate(ctx, params)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("missing or invalid capability param"))
		})
		It("returns err when no PaasConfig is set", func() {
			By("Calling Generate")
			err := k8sClient.Delete(ctx, &conf)
			Expect(err).To(Not(HaveOccurred()))

			params := fields.ElementMap{
				"capability": "argocd",
			}
			results, err := svc.Generate(ctx, params)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no PaasConfig found"))
			Expect(results).To(BeEmpty())
		})

		It("generates elements even when custom fields don't match validation regex", func() {
			By("Creating a Paas with custom fields that would fail validation")

			// This git_url doesn't match the validation regex "^ssh://git@scm/[a-zA-Z0-9-./]*.git$"
			// but the plugin generator should still return the values
			invalidGitURL := "https://github.com/invalid/repo.git"
			paas := &v1alpha2.Paas{
				ObjectMeta: metav1.ObjectMeta{
					Name: "paas-with-invalid-fields",
				},
				Spec: v1alpha2.PaasSpec{
					Requestor: "test-requestor",
					Quota:     quota.Quota{},
					Capabilities: map[string]v1alpha2.PaasCapability{
						"argocd": {
							CustomFields: map[string]string{
								"git_url":      invalidGitURL,
								"git_path":     "path/",
								"git_revision": "main",
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, paas)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, paas)).To(Succeed())
			})

			By("Calling Generate - it should succeed without validation errors")

			params := fields.ElementMap{
				"capability": "argocd",
			}
			results, err := svc.Generate(ctx, params)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).NotTo(BeEmpty())

			// Find our paas in results
			var foundPaas fields.ElementMap
			for _, r := range results {
				if r["paas"] == "paas-with-invalid-fields" {
					foundPaas = r
					break
				}
			}
			Expect(foundPaas).NotTo(BeNil())
			Expect(foundPaas["git_url"]).To(Equal(invalidGitURL))
		})
	})
})

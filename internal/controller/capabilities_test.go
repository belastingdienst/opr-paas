/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/config"
	"github.com/belastingdienst/opr-paas/v3/internal/fields"
	appv1 "github.com/belastingdienst/opr-paas/v3/internal/stubs/argoproj/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Capabilities controller", Ordered, func() {
	const (
		serviceName        = "my"
		capName            = serviceName
		capAppSetName      = capName + "-as"
		capAppSetNamespace = "asns"
		paasName           = capName + "-paas"
		invalidCapAppSet   = "does-not-exist"
		customField1Key    = "custom field 1"
		customField1Value  = "custom value 1"
		customField2Key    = "custom field 2"
		customField2Value  = "custom value 2"
		group1             = "ldapgroup"
		group1Query        = "CN=group1OU=example"
		group2             = "usergroup"
	)

	var (
		ctx         context.Context
		paas        *v1alpha2.Paas
		reconciler  *PaasReconciler
		paasConfig  v1alpha2.PaasConfig
		group1Roles = []string{"admin"}
		group2Users = []string{"user1", "user2"}
		group2Roles = []string{"edit", "view"}
	)

	BeforeAll(func() {
		paas = &v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
				UID:  "abc", // Needed or owner references fail
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: capName,
				Capabilities: v1alpha2.PaasCapabilities{
					capName: v1alpha2.PaasCapability{
						CustomFields: map[string]string{
							customField1Key: customField1Value,
							customField2Key: customField2Value,
						},
					},
				},
				Groups: v1alpha2.PaasGroups{
					group1: v1alpha2.PaasGroup{Query: group1Query, Roles: group1Roles},
					group2: v1alpha2.PaasGroup{Users: group2Users, Roles: group2Roles},
				},
			},
		}
	})

	BeforeEach(func() {
		const (
			groupTemplate = `g, system:cluster-admins, role:admin{{ range $groupName, $group := .Paas.Spec.Groups }}
g, {{ $groupName }}, role:admin{{end}}`
		)
		paasConfig = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: v1alpha2.PaasConfigSpec{
				ClusterWideArgoCDNamespace: capAppSetNamespace,
				Capabilities: map[string]v1alpha2.ConfigCapability{
					capName: {
						AppSet: capAppSetName,
						CustomFields: map[string]v1alpha2.ConfigCustomField{
							customField1Key: {},
							customField2Key: {},
							"argocd_default_policy": {
								Default: "",
							},
							"argocd_policy": {
								Template: groupTemplate,
							},
							"argocd_scopes": {
								Default: "[groups]",
							},
						},
					},
				},
				Templating: v1alpha2.ConfigTemplatingItems{
					GenericCapabilityFields: v1alpha2.ConfigTemplatingItem{
						"requestor":  "{{ .Paas.Spec.Requestor }}",
						"service":    "{{ (splitn \"-\" 2 .Paas.Name)._0 }}",
						"subservice": "{{ (splitn \"-\" 2 .Paas.Name)._1 }}",
					},
				},
			},
		}

		// Updates context to include paasConfig
		ctx = context.WithValue(context.Background(), config.ContextKeyPaasConfig, paasConfig)

		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		assureAppSet(ctx, capAppSetName, capAppSetNamespace)
	})

	AfterEach(func() {
		appset := &appv1.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      capAppSetName,
				Namespace: capAppSetNamespace,
			},
		}
		err := k8sClient.Delete(ctx, appset)
		Expect(err).NotTo(HaveOccurred())
	})

	When("ensuring capability in the AppSet", func() {
		Context("with a valid capability configuration", Ordered, func() {
			appSetName := types.NamespacedName{
				Name:      capAppSetName,
				Namespace: capAppSetNamespace,
			}

			It("should succeed", func() {
				err := reconciler.ensureAppSetCap(ctx, paas, capName)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should create an appset entry with proper data", func() {
				const (
					expectedPolicy = `g, system:cluster-admins, role:admin
g, ` + group1 + `, role:admin
g, ` + group2 + `, role:admin`
				)
				err := reconciler.ensureAppSetCap(ctx, paas, capName)
				Expect(err).NotTo(HaveOccurred())
				appSet := &appv1.ApplicationSet{}
				err = k8sClient.Get(ctx, appSetName, appSet)
				Expect(err).NotTo(HaveOccurred())
				entries := make(fields.Entries)
				for _, generator := range appSet.Spec.Generators {
					var generatorEntries fields.Entries
					generatorEntries, err = fields.EntriesFromJSON(generator.List.Elements)
					Expect(err).NotTo(HaveOccurred())
					entries = entries.Merge(generatorEntries)
				}
				Expect(entries).To(HaveKey(paasName))
				elements := entries[paasName]
				Expect(elements).To(Equal(
					fields.ElementMap{
						customField1Key:         customField1Value,
						customField2Key:         customField2Value,
						"paas":                  paasName,
						"argocd_default_policy": "",
						"argocd_policy":         expectedPolicy,
						"argocd_scopes":         "[groups]",
						"requestor":             "my",
						"service":               serviceName,
						"subservice":            "paas",
					}))
			})
			It("should delete the appset entry during finalization", func() {
				appSet := &appv1.ApplicationSet{}
				Expect(reconciler.ensureAppSetCap(ctx, paas, capName)).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, appSetName, appSet)).NotTo(HaveOccurred())
				Expect(appSet.Spec.Generators[0].List.Elements).To(HaveLen(1))

				Expect(reconciler.finalizeAllAppSetCaps(ctx, paas)).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, appSetName, appSet)).NotTo(HaveOccurred())
				Expect(appSet.Spec.Generators[0].List.Elements).To(BeEmpty())
			})
		})

		Context("without pointing to a proper AppSet", func() {
			It("should fail", func() {
				argoCapConfig := paasConfig.Spec.Capabilities[capName]
				argoCapConfig.AppSet = invalidCapAppSet
				paasConfig.Spec.Capabilities[capName] = argoCapConfig
				// Updates context with updated PaasConfig
				ctx = context.WithValue(ctx, config.ContextKeyPaasConfig, paasConfig)
				err := reconciler.ensureAppSetCap(ctx, paas, capName)
				Expect(err).Error().To(MatchError(
					ContainSubstring("applicationsets.argoproj.io \"" + invalidCapAppSet + "\" not found")))
			})
		})
	})
})

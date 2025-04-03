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
	"github.com/belastingdienst/opr-paas/internal/fields"
	appv1 "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Capabilities controller", Ordered, func() {
	const (
		serviceName        = "my"
		subServiceName     = "cap"
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
		paas        *api.Paas
		reconciler  *PaasReconciler
		appSet      *appv1.ApplicationSet
		paasConfig  api.PaasConfig
		group1Roles = []string{"admin"}
		group2Users = []string{"user1", "user2"}
		group2Roles = []string{"edit", "view"}
	)

	BeforeAll(func() {
		paas = &api.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
				UID:  "abc", // Needed or owner references fail
			},
			Spec: api.PaasSpec{
				Requestor: capName,
				Capabilities: api.PaasCapabilities{
					capName: api.PaasCapability{
						Enabled: true,
						CustomFields: map[string]string{
							customField1Key: customField1Value,
							customField2Key: customField2Value,
						},
					},
				},
				Groups: api.PaasGroups{
					group1: api.PaasGroup{Query: group1Query, Roles: group1Roles},
					group2: api.PaasGroup{Users: group2Users, Roles: group2Roles},
				},
			},
		}
	})

	BeforeEach(func() {
		const (
			groupTemplate = `g, system:cluster-admins, role:admin{{ range $groupName, $group := .Paas.Spec.Groups }}
g, {{ $groupName }}, role:admin{{end}}`
		)
		ctx = context.Background()
		paasConfig = api.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: api.PaasConfigSpec{
				ClusterWideArgoCDNamespace: capAppSetNamespace,
				Capabilities: map[string]api.ConfigCapability{
					capName: {
						AppSet: capAppSetName,
						CustomFields: map[string]api.ConfigCustomField{
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
			},
		}
		config.SetConfig(paasConfig)
		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		appSet = &appv1.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      capAppSetName,
				Namespace: capAppSetNamespace,
			},
			Spec: appv1.ApplicationSetSpec{
				Generators: []appv1.ApplicationSetGenerator{},
			},
		}
		err := k8sClient.Create(ctx, appSet)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := k8sClient.Delete(ctx, appSet)
		Expect(err).NotTo(HaveOccurred())
	})

	When("ensuring capability in the AppSet", func() {
		Context("with a valid capability configuration", Ordered, func() {
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
				appSet = &appv1.ApplicationSet{}
				appSetName := types.NamespacedName{
					Name:      capAppSetName,
					Namespace: capAppSetNamespace,
				}
				err = k8sClient.Get(ctx, appSetName, appSet)
				Expect(err).NotTo(HaveOccurred())
				entries := make(fields.Entries)
				for _, generator := range appSet.Spec.Generators {
					generatorEntries, err := fields.EntriesFromJSON(generator.List.Elements)
					Expect(err).NotTo(HaveOccurred())
					entries = entries.Merge(generatorEntries)
				}
				Expect(entries).To(HaveKey(paasName))
				elements := entries[paasName]
				Expect(elements.GetElementsAsStringMap()).To(Equal(
					map[string]string{
						customField1Key:         customField1Value,
						customField2Key:         customField2Value,
						"git_path":              "",
						"git_revision":          "",
						"git_url":               "",
						"paas":                  paasName,
						"argocd_default_policy": "",
						"argocd_policy":         expectedPolicy,
						"argocd_scopes":         "[groups]",
						"requestor":             "my",
						"service":               serviceName,
						"subservice":            "paas",
					}))
			})
		})

		Context("without pointing to a proper AppSet", func() {
			It("should fail", func() {
				argoCapConfig := paasConfig.Spec.Capabilities[capName]
				argoCapConfig.AppSet = invalidCapAppSet
				paasConfig.Spec.Capabilities[capName] = argoCapConfig
				config.SetConfig(paasConfig)
				err := reconciler.ensureAppSetCap(ctx, paas, capName)
				Expect(err).Error().To(MatchError(
					ContainSubstring("applicationsets.argoproj.io \"" + invalidCapAppSet + "\" not found")))
			})
		})
	})
})

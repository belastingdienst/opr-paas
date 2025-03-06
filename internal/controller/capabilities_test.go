/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"
	"reflect"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/fields"
	appv1 "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func groupFromInterface(iface interface{}) (group api.PaasGroup, err error) {
	switch v := reflect.ValueOf(iface); v.Kind() {
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			value := iter.Value()
			if value.Kind() == reflect.Interface {
				value = reflect.ValueOf(value.Interface())
			}
			switch key := iter.Key().String(); key {
			case "query":
				group.Query = value.String()
			case "roles", "users":
				var stringList []string
				if value.Kind() != reflect.Slice {
					return api.PaasGroup{},
						fmt.Errorf("%s value is not a slice (%d), %s", key, value.Kind(), value.String())
				}
				for i := 0; i < value.Len(); i++ {
					subValue := value.Index(i)
					if subValue.Kind() == reflect.Interface {
						subValue = reflect.ValueOf(subValue.Interface())
					}
					stringList = append(stringList, subValue.String())
				}
				if key == "roles" {
					group.Roles = stringList
				} else {
					group.Users = stringList
				}
			default:
				return api.PaasGroup{}, fmt.Errorf("unexpected field in group: %s", key)
			}
		}
	default:
		return api.PaasGroup{}, fmt.Errorf("input is not a map (%s)", v.String())
	}
	return group, nil
}

func groupsFromInterface(iface interface{}) (groups api.PaasGroups, err error) {
	groups = make(api.PaasGroups)
	switch v := reflect.ValueOf(iface); v.Kind() {
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			group, err := groupFromInterface(iter.Value().Interface())
			if err != nil {
				return nil, fmt.Errorf("groups value cannot be converted to group: %w", err)
			}
			groups[key] = group
		}
	default:
		return nil, fmt.Errorf("input is not a map: %s", v.String())
	}
	return groups, nil
}

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
						customField1Key: customField1Value,
						customField2Key: customField2Value,
						"git_path":      "",
						"git_revision":  "",
						"git_url":       "",
						"groups":        "",
						"paas":          paasName,
						"requestor":     "my",
						"service":       serviceName,
						"subservice":    "paas",
					}))
				groupsObj, ok := elements["groups"]
				Expect(ok).To(BeTrue())

				groups, err := groupsFromInterface(groupsObj)
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(Equal(paas.Spec.Groups))
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

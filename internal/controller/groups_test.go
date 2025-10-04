/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	userv1 "github.com/openshift/api/user/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Group controller", Ordered, func() {
	const (
		lbl1Key       = "key1"
		lbl1Value     = "value1"
		lbl2Key       = "key2"
		lbl2Value     = "value2"
		kubeInstLabel = "app.kubernetes.io/instance"
	)
	var (
		ctx        context.Context
		paas       *v1alpha2.Paas
		myConfig   v1alpha2.PaasConfig
		group      *userv1.Group
		reconciler *PaasReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		group = &userv1.Group{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-group",
			},
		}
		// Set the PaasConfig so reconcilers know where to find our fixtures

		myConfig = v1alpha2.PaasConfig{
			Spec: v1alpha2.PaasConfigSpec{
				DecryptKeysSecret: v1alpha2.NamespacedName{
					Name:      "keys",
					Namespace: "paas-system",
				},
				Templating: v1alpha2.ConfigTemplatingItems{
					GroupLabels: map[string]string{
						"": "{{ range $key, $value := .Paas.Labels }}{{ if ne $key \"" +
							kubeInstLabel + "\" }}{{$key}}: {{$value}}\n{{end}}{{end}}",
					},
				},
			},
		}

		// Updates context to include paasConfig
		ctx = context.WithValue(context.Background(), contextKeyPaasConfig, myConfig)
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, group)
	})

	Context("When creating a paas with users", func() {
		paas = &v1alpha2.Paas{ObjectMeta: metav1.ObjectMeta{
			Name: "my-paas",
			UID:  "abc", // Needed or owner references fail
			Labels: map[string]string{
				lbl1Key:       lbl1Value,
				lbl2Key:       lbl2Value,
				kubeInstLabel: "whatever",
			},
		}}
		It("should create the group if it does not exist", func() {
			group.Users = []string{"hank", "pete"}
			err := reconciler.ensureGroup(ctx, paas, group)
			Expect(err).NotTo(HaveOccurred())

			found := &userv1.Group{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: group.Name}, found)
			Expect(err).NotTo(HaveOccurred())
			Expect(found.Users).To(Equal(group.Users))
		})

		It("should update the group if users list changes", func() {
			// Create the group first
			err := k8sClient.Create(ctx, group)
			Expect(err).NotTo(HaveOccurred())

			// Modify users
			group.Users = []string{"user1", "user2"}

			err = reconciler.ensureGroup(ctx, paas, group)
			Expect(err).NotTo(HaveOccurred())

			updated := &userv1.Group{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: group.Name}, updated)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Users).To(Equal(group.Users))
		})

		It("should set the owner reference if not already set", func() {
			// Create group without owner reference
			err := k8sClient.Create(ctx, group)
			Expect(err).NotTo(HaveOccurred())
			Expect(group.OwnerReferences).To(BeEmpty())

			err = reconciler.ensureGroup(ctx, paas, group)
			Expect(err).NotTo(HaveOccurred())

			updated := &userv1.Group{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: group.Name}, updated)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.OwnerReferences).NotTo(BeEmpty())
			Expect(updated.OwnerReferences[0].UID).To(Equal(paas.UID))
		})

		It("have set all expected labels", func() {
			var (
				expectedLabels = map[string]string{
					lbl1Key:           lbl1Value,
					lbl2Key:           lbl2Value,
					ManagedByLabelKey: paas.Name,
				}
			)
			paas.Spec.Groups = v1alpha2.PaasGroups{
				group.Name: v1alpha2.PaasGroup{Users: []string{"u1", "u2"}},
			}
			err := reconciler.reconcileGroups(ctx, paas)
			Expect(err).NotTo(HaveOccurred())
			groups, err := reconciler.backendGroups(ctx, paas)
			Expect(err).NotTo(HaveOccurred())
			Expect(groups).To(HaveLen(1))

			for _, group := range groups {
				err = k8sClient.Get(ctx, types.NamespacedName{Name: group.Name}, group)
				Expect(err).NotTo(HaveOccurred())
				for key, value := range expectedLabels {
					Expect(group.ObjectMeta.Labels).To(HaveKeyWithValue(key, value))
				}
				Expect(group.ObjectMeta.Labels).NotTo(HaveKey(kubeInstLabel))
			}
		})
	})
	Context("When using feature flags", func() {
		It("should allow, warn or succeed when groups have users", func() {
			for setting, expects := range map[string]struct {
				groups bool
			}{
				"":      {groups: true},
				"allow": {groups: true},
				"warn":  {groups: true},
				"block": {},
			} {
				paas = &v1alpha2.Paas{ObjectMeta: metav1.ObjectMeta{
					Name: "my-paas",
					UID:  "abc", // Needed or owner references fail
				},
					Spec: v1alpha2.PaasSpec{
						Groups: v1alpha2.PaasGroups{
							"some_group": v1alpha2.PaasGroup{
								Users: []string{"usr1", "usr2"},
							},
						},
					},
				}
				fmt.Fprintf(GinkgoWriter, "DEBUG - Test: %s: %v", setting, expects)
				myConfig.Spec.FeatureFlags.GroupUserManagement = setting

				// Updates context to include paasConfig
				ctx = context.WithValue(context.Background(), contextKeyPaasConfig, myConfig)

				groups, err := reconciler.backendGroups(ctx, paas)
				Expect(err).NotTo(HaveOccurred())
				if expects.groups {
					Expect(groups).NotTo(BeEmpty())
				} else {
					Expect(groups).To(BeEmpty())
				}
			}
		})
	})
})

/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	userv1 "github.com/openshift/api/user/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Group controller", Ordered, func() {
	var (
		ctx        context.Context
		paas       *api.Paas
		group      *userv1.Group
		reconciler *PaasReconciler
	)

	BeforeAll(func() {
		paas = &api.Paas{ObjectMeta: metav1.ObjectMeta{
			Name: "my-paas",
			UID:  "abc", // Needed or owner references fail
		}}
	})

	BeforeEach(func() {
		ctx = context.Background()
		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		group = &userv1.Group{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-group",
				Namespace: "default",
			},
		}
	})

	AfterEach(func() {
		err := k8sClient.Delete(ctx, group)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create the group if it does not exist", func() {
		group.Users = []string{"hank", "pete"}
		err := reconciler.EnsureGroup(ctx, paas, group)
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

		err = reconciler.EnsureGroup(ctx, paas, group)
		Expect(err).NotTo(HaveOccurred())

		updated := &userv1.Group{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: group.Name}, updated)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Users).To(Equal(group.Users))
	})

	It("should not update the group if only users list changes and it is an ldap group", func() {
		// ldap managed group has a label and users (from ldap)
		initialUsers := userv1.OptionalNames([]string{"user1", "user2"})
		changedUsers := userv1.OptionalNames([]string{"us", "them"})
		group.Labels = map[string]string{ldapHostLabelKey: "somehost"}
		group.Users = initialUsers

		// Create the group
		err := k8sClient.Create(ctx, group)
		Expect(err).NotTo(HaveOccurred())

		// Modify users
		group.Users = changedUsers

		err = reconciler.EnsureGroup(ctx, paas, group)
		Expect(err).NotTo(HaveOccurred())

		updated := &userv1.Group{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: group.Name}, updated)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Users).NotTo(Equal(group.Users))
		Expect(updated.Users).To(Equal(initialUsers))
	})

	It("should set the owner reference if not already set", func() {
		// Create group without owner reference
		err := k8sClient.Create(ctx, group)
		Expect(err).NotTo(HaveOccurred())
		Expect(group.OwnerReferences).To(BeEmpty())

		err = reconciler.EnsureGroup(ctx, paas, group)
		Expect(err).NotTo(HaveOccurred())

		updated := &userv1.Group{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: group.Name}, updated)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.OwnerReferences).NotTo(BeEmpty())
		Expect(updated.OwnerReferences[0].UID).To(Equal(paas.UID))
	})
})

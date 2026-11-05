/*
Copyright 2026, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

import (
	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("validateGroupNameCollisions", func() {
	const (
		queryTeamA = "CN=admins,OU=team-a,DC=example,DC=org"
		queryTeamB = "CN=admins,OU=team-b,DC=example,DC=org"
	)
	paasWithGroups := func(name string, groups v1alpha2.PaasGroups) *v1alpha2.Paas {
		return &v1alpha2.Paas{ObjectMeta: metav1.ObjectMeta{Name: name}, Spec: v1alpha2.PaasSpec{Groups: groups}}
	}
	validate := func(oldPaas, paas *v1alpha2.Paas) []string {
		fakeClient := fake.NewClientBuilder().
			WithScheme(k8sClient.Scheme()).
			WithObjects(
				paasWithGroups("existing-a", v1alpha2.PaasGroups{"admins": {Query: queryTeamA}}),
				paasWithGroups("existing-b", v1alpha2.PaasGroups{"devs": {Users: []string{"user1"}}}),
			).
			Build()
		errs, err := validateGroupNameCollisions(oldPaas)(ctx, fakeClient, v1alpha2.PaasConfig{}, paas)
		Expect(err).NotTo(HaveOccurred())
		var messages []string
		for _, e := range errs {
			messages = append(messages, e.Error())
		}
		return messages
	}

	It("rejects a query with the same CN but a different LDAP group", func() {
		errs := validate(nil, paasWithGroups("new-paas", v1alpha2.PaasGroups{"admins": {Query: queryTeamB}}))
		Expect(errs).To(ConsistOf(ContainSubstring("group name admins is already used by Paas existing-a")))
	})

	It("allows the same query in multiple Paases (the same LDAP group)", func() {
		Expect(validate(nil, paasWithGroups("new-paas", v1alpha2.PaasGroups{"admins": {Query: queryTeamA}}))).
			To(BeEmpty())
	})

	It("rejects a query whose CN equals the name of a user group of another Paas", func() {
		// Paas existing-b has a user group named existing-b-devs
		errs := validate(nil, paasWithGroups("new-paas", v1alpha2.PaasGroups{
			"devs": {Query: "CN=existing-b-devs,OU=x,DC=example,DC=org"},
		}))
		Expect(errs).To(ConsistOf(ContainSubstring("already used by Paas existing-b")))
	})

	It("does not check unchanged groups on update, so existing collisions do not block other changes", func() {
		colliding := paasWithGroups("new-paas", v1alpha2.PaasGroups{"admins": {Query: queryTeamB}})
		Expect(validate(colliding, colliding.DeepCopy())).To(BeEmpty())
	})

	It("checks groups whose query changed on update", func() {
		oldPaas := paasWithGroups("new-paas", v1alpha2.PaasGroups{"admins": {Query: "CN=other,OU=x,DC=example,DC=org"}})
		newPaas := paasWithGroups("new-paas", v1alpha2.PaasGroups{"admins": {Query: queryTeamB}})
		Expect(validate(oldPaas, newPaas)).To(HaveLen(1))
	})
})

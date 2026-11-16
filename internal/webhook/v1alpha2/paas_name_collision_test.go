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

var _ = Describe("validatePaasNameCollisions", func() {
	existing := func(name string) *v1alpha2.Paas {
		return &v1alpha2.Paas{ObjectMeta: metav1.ObjectMeta{Name: name}}
	}

	DescribeTable("checks a new Paas name against existing Paases",
		func(newName string, collides bool) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(k8sClient.Scheme()).
				WithObjects(existing("team"), existing("other-app")).
				Build()

			errs, err := validatePaasNameCollisions(ctx, fakeClient, v1alpha2.PaasConfig{}, existing(newName))
			Expect(err).NotTo(HaveOccurred())
			if collides {
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Error()).To(ContainSubstring("name collides with existing Paas"))
			} else {
				Expect(errs).To(BeEmpty())
			}
		},
		Entry("existing name is a prefix of the new name", "team-x", true),
		Entry("new name is a prefix of an existing name", "other", true),
		Entry("same name (the Paas itself)", "team", false),
		Entry("shared prefix without a dash boundary", "teamx", false),
		Entry("shorter name without a dash boundary", "tea", false),
		Entry("unrelated name", "something", false),
	)
})

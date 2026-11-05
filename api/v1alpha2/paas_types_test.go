package v1alpha2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
)

var _ = Describe("PaasTypes", func() {
	var paas *v1alpha2.Paas
	const paasName = "mypaas"
	BeforeEach(func() {
		paas = &v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
			},
		}
	})
	Describe("New Paas", func() {
		Context("with default values", func() {
			It("should have name properly set", func() {
				Expect(paas.Name).To(Equal(paasName))
			})
		})
	})

	Describe("CapExtraFields", func() {
		// An empty validation regex matches anything, so only the built-in control character check applies.
		fieldConfig := map[string]v1alpha2.ConfigCustomField{
			"git_revision": {},
		}

		DescribeTable("rejects control characters when validating",
			func(value string) {
				capability := v1alpha2.PaasCapability{CustomFields: map[string]string{"git_revision": value}}
				elements, err := capability.CapExtraFields(fieldConfig, true)
				Expect(err).To(MatchError(ContainSubstring("custom field git_revision contains control characters")))
				Expect(elements).To(BeNil())
			},
			Entry("YAML document injection", "main\n---\napiVersion: v1\nkind: ConfigMap"),
			Entry("carriage return", "main\rfoo"),
			Entry("tab", "main\tfoo"),
			Entry("NUL", "main\x00"),
		)

		It("accepts a regular single-line value", func() {
			capability := v1alpha2.PaasCapability{CustomFields: map[string]string{"git_revision": "feature/my-branch"}}
			elements, err := capability.CapExtraFields(fieldConfig, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(elements).To(HaveKeyWithValue("git_revision", "feature/my-branch"))
		})
	})
})

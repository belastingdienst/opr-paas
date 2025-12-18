package v1alpha2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
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
})

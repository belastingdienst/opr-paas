/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	apiv1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Paas Webhook", func() {
	var (
		conf      *apiv1alpha1.PaasConfig
		obj       *apiv1alpha1.Paas
		oldObj    *apiv1alpha1.Paas
		validator PaasCustomValidator
	)

	BeforeEach(func() {
		obj = &apiv1alpha1.Paas{}
		oldObj = &apiv1alpha1.Paas{}
		validator = PaasCustomValidator{k8sClient}
		conf = &apiv1alpha1.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "conf", Namespace: "paas-system"},
			Spec: apiv1alpha1.PaasConfigSpec{
				DecryptKeysSecret:          apiv1alpha1.NamespacedName{"keys", "paas-system"},
				Capabilities:               apiv1alpha1.ConfigCapabilities{},
				GroupSyncList:              apiv1alpha1.NamespacedName{"wlname", "gsns"},
				ExcludeAppSetName:          "whatever",
				LDAP:                       apiv1alpha1.ConfigLdap{Host: "some-ldap-host", Port: 13},
				ArgoPermissions:            apiv1alpha1.ConfigArgoPermissions{ResourceName: "argocd", Header: "g", Role: "admin"},
				ClusterWideArgoCDNamespace: "asns",
			},
		}

		Expect(k8sClient.Create(ctx, conf)).NotTo(HaveOccurred())
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, conf)).NotTo(HaveOccurred())
	})

	Context("When creating a Paas under Validating Webhook", func() {
		It("Should deny creation when an unconfigured capability is set", func() {
			obj = &apiv1alpha1.Paas{
				Spec: apiv1alpha1.PaasSpec{
					Capabilities: apiv1alpha1.PaasCapabilities{"foo": apiv1alpha1.PaasCapability{}},
				},
			}

			Expect(validator.ValidateCreate(ctx, obj)).Error().To(MatchError("capability foo not configured"))
		})

		//It("Should admit creation if all required fields are present", func() {
		//	By("simulating an invalid creation scenario")
		//	Expect(validator.ValidateCreate(ctx, obj)).To(BeNil())
		//})
		//
		// It("Should validate updates correctly", func() {
		//     By("simulating a valid update scenario")
		//     oldObj.SomeRequiredField = "updated_value"
		//     obj.SomeRequiredField = "updated_value"
		//     Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil())
		// })
	})
})

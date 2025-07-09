package controller

import (
	"context"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/config"
	"github.com/belastingdienst/opr-paas/v3/pkg/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("NamespaceDef", func() {
	const (
		enabledCapName   = "enabled-cap1"
		disabledCapName1 = "disabled-cap1"
		disabledCapName2 = "disabled-cap2"
		paasName         = "my-paas"
		ns1              = "ns1"
		ns2              = "ns2"
		group1           = "g1"
		group1Query      = "CN=" + group1 + ",OU=paas,DC=test,DC=acme,DC=org"
		group2           = "g2"
		group3           = "g3"
	)
	var (
		paas       v1alpha2.Paas
		paasConfig v1alpha2.PaasConfig
		ctx        context.Context
		reconciler *PaasReconciler
		namespaces = v1alpha2.PaasNamespaces{
			ns1: v1alpha2.PaasNamespace{},
			ns2: v1alpha2.PaasNamespace{},
		}
		paasNsGroups = []string{group1, group2}
	)
	BeforeEach(func() {
		ctx = context.Background()
		paas = v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: "somebody",
				Capabilities: v1alpha2.PaasCapabilities{
					enabledCapName: v1alpha2.PaasCapability{},
				},
				Namespaces: namespaces,
				Groups: v1alpha2.PaasGroups{
					group1: v1alpha2.PaasGroup{Query: group1Query},
					group2: v1alpha2.PaasGroup{Users: []string{"usr2"}},
					group3: v1alpha2.PaasGroup{Users: []string{"usr3"}},
				},
				Quota: quota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
				Secrets: map[string]string{
					"default-secret": "default-value",
				},
			},
		}
		assurePaas(ctx, paas)
		paasConfig = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: v1alpha2.PaasConfigSpec{
				Capabilities: map[string]v1alpha2.ConfigCapability{
					enabledCapName:   {},
					disabledCapName1: {},
					disabledCapName2: {},
				},
			},
		}
		config.SetConfig(paasConfig)
		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	When("getting a nsdef from a paas", func() {
		Context("with any paas", func() {
			var nsDefs namespaceDefs
			It("should succeed", func() {
				var err error
				nsDefs, err = reconciler.nsDefsFromPaas(ctx, &paas)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not return a nsdef for a ns named after the paas", func() {
				Expect(nsDefs).ToNot(HaveKey(paasName))
			})
		})
		Context("with some capabilities enabled and others disabled", func() {
			var nsDefs namespaceDefs
			It("should succeed", func() {
				var err error
				nsDefs, err = reconciler.nsDefsFromPaas(ctx, &paas)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return nsdefs for enabled capabilities", func() {
				Expect(nsDefs).To(HaveKey(join(paasName, enabledCapName)))
			})
			It("should not return nsdefs for disabled capabilities", func() {
				Expect(nsDefs).NotTo(HaveKey(join(paasName, disabledCapName1)))
				Expect(nsDefs).NotTo(HaveKey(join(paasName, disabledCapName2)))
			})
		})
		Context("with namespaces in the namespace block", func() {
			var nsDefs namespaceDefs
			It("should succeed", func() {
				var err error
				nsDefs, err = reconciler.nsDefsFromPaas(ctx, &paas)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return nsdefs for every defined namespace", func() {
				for nsName := range nsDefs {
					Expect(nsDefs).To(HaveKey(nsName))
				}
			})
		})
		Context("with paasns objects", func() {
			var nsDefs namespaceDefs
			It("should successfully create paasns's", func() {
				myPnss := map[string]struct {
					namespace string
					groups    []string
				}{
					ns1:              {namespace: join(paasName, ns1)},
					enabledCapName:   {namespace: join(paasName, enabledCapName)},
					disabledCapName1: {namespace: join(paasName, disabledCapName1)},
					disabledCapName2: {namespace: join(paasName, disabledCapName1)},
					"recursive":      {namespace: join(paasName, "mypns", ns1)},
				}
				for pnsName, pnsDef := range myPnss {
					assureNamespaceWithPaasReference(ctx, pnsDef.namespace, paasName)
					pns := v1alpha2.PaasNS{
						ObjectMeta: metav1.ObjectMeta{
							Name:      join("mypns", pnsName),
							Namespace: pnsDef.namespace,
						},
						Spec: v1alpha2.PaasNSSpec{
							Paas:   paasName,
							Groups: pnsDef.groups,
						},
					}
					err := reconciler.Create(ctx, &pns)
					Expect(err).NotTo(HaveOccurred())
					validatePaasNSExists(ctx, pnsDef.namespace, join("mypns", pnsName))
				}
			})
			It("should succeed", func() {
				var err error
				nsDefs, err = reconciler.nsDefsFromPaas(ctx, &paas)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return a nsdef for every paasns in a namespace block ns", func() {
				Expect(nsDefs).To(HaveKey(join(paasName, "mypns", ns1)))
			})
			It("should return a nsdef for every paasns in an enabled cap ns", func() {
				Expect(nsDefs).To(HaveKey(join(paasName, "mypns", enabledCapName)))
			})
			It("should not return a nsdef for anyy paasns in a disabled cap ns", func() {
				Expect(nsDefs).NotTo(HaveKey(join(paasName, "mypns", disabledCapName1)))
				Expect(nsDefs).NotTo(HaveKey(join(paasName, "mypns", disabledCapName2)))
			})
			It("should return a nsdef for every nested paasns object", func() {
				Expect(nsDefs).To(HaveKey(join(paasName, "mypns", "recursive")))
			})
		})
		Context("with a paas with groups", func() {
			var nsDefs namespaceDefs
			It("should successfully create paasns's", func() {
				assureNamespaceWithPaasReference(ctx, paasName, paasName)
				myPnss := map[string]struct {
					namespace string
					groups    []string
				}{
					"nongroups": {namespace: join(paasName, ns1)},
					"groups":    {namespace: join(paasName, ns1), groups: paasNsGroups},
				}
				for pnsName, pnsDef := range myPnss {
					assureNamespaceWithPaasReference(ctx, pnsDef.namespace, paasName)
					pns := v1alpha2.PaasNS{
						ObjectMeta: metav1.ObjectMeta{
							Name:      pnsName,
							Namespace: pnsDef.namespace,
						},
						Spec: v1alpha2.PaasNSSpec{
							Paas:   paasName,
							Groups: pnsDef.groups,
						},
					}
					err := reconciler.Create(ctx, &pns)
					Expect(err).NotTo(HaveOccurred())
					validatePaasNSExists(ctx, join(paasName, ns1), pnsName)
				}
			})
			It("should succeed", func() {
				var err error
				nsDefs, err = reconciler.nsDefsFromPaas(ctx, &paas)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return a nsdef with proper group permissions for ldap queries", func() {
				nsName := join(paasName, "nongroups")
				Expect(nsDefs).To(HaveKey(nsName))
				withoutGroups := nsDefs[nsName].groups
				Expect(withoutGroups).To(ContainElement(group1))
				Expect(withoutGroups).NotTo(ContainElement(join(paasName, group1)))
			})
			It("should return a nsdef with proper group permissions for non-ldap queries", func() {
				nsName := join(paasName, "nongroups")
				Expect(nsDefs).To(HaveKey(nsName))
				withoutGroups := nsDefs[nsName].groups
				Expect(withoutGroups).To(ContainElement(group2))
				Expect(withoutGroups).To(ContainElement(group2))
				Expect(withoutGroups).To(ContainElement(group3))
			})
			It("should have proper group config if paasns has group config set", func() {
				nsName := join(paasName, "groups")
				Expect(nsDefs).To(HaveKey(nsName))
				withGroups := nsDefs[nsName].groups
				Expect(withGroups).To(ContainElement(group1))
				Expect(withGroups).NotTo(ContainElement(join(paasName, group1)))
				Expect(withGroups).To(ContainElement(group2))
				Expect(withGroups).NotTo(ContainElement(join(paasName, group2)))
			})
		})
		Context("with secrets defined in paas and paasns", func() {
			var nsDefs namespaceDefs
			var pns v1alpha2.PaasNS
			const paasNsName string = "secret"

			BeforeEach(func() {
				// Create a PaasNS with its own secrets
				nsName := join(paasName, ns1)
				assureNamespaceWithPaasReference(ctx, nsName, paasName)
				pns = v1alpha2.PaasNS{
					ObjectMeta: metav1.ObjectMeta{
						Name:      paasNsName,
						Namespace: nsName,
					},
					Spec: v1alpha2.PaasNSSpec{
						Paas: paasName,
						Secrets: map[string]string{
							"pns-secret":     "pns-value",
							"default-secret": "overridden-value",
						},
					},
				}
				err := reconciler.Create(ctx, &pns)
				Expect(err).NotTo(HaveOccurred())
				validatePaasNSExists(ctx, nsName, paasNsName)
			})
			AfterEach(func() {
				_ = reconciler.Delete(ctx, &pns)
			})
			It("should succeed", func() {
				var err error
				nsDefs, err = reconciler.nsDefsFromPaas(ctx, &paas)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should include default secrets in paas namespace", func() {
				ns := nsDefs[join(paasName, ns1)]
				Expect(ns.secrets).To(HaveKeyWithValue("default-secret", "default-value"))
			})
			It("should include paasns secrets in paasns namespace def", func() {
				ns := nsDefs[join(paasName, paasNsName)]
				Expect(ns.secrets).To(HaveKeyWithValue("pns-secret", "pns-value"))
				Expect(ns.secrets).To(HaveKeyWithValue("default-secret", "overridden-value"))
			})
		})
	})
})

package controller

import (
	"context"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func validatePaasNSExists(ctx context.Context, namespaceName string, paasNSName string) {
	pns := api.PaasNS{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: paasNSName, Namespace: namespaceName}, &pns)
	Expect(err).NotTo(HaveOccurred())
}

func assureNamespaceWithPaasReference(ctx context.Context, namespaceName string, paasName string) {
	assureNamespace(ctx, namespaceName)
	paas := &api.Paas{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: paasName}, paas)
	Expect(err).NotTo(HaveOccurred())
	ns := &corev1.Namespace{}
	err = k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, ns)
	Expect(err).NotTo(HaveOccurred())

	if !paas.AmIOwner(ns.GetOwnerReferences()) {
		patchedNs := client.MergeFrom(ns.DeepCopy())
		controllerutil.SetControllerReference(paas, ns, scheme.Scheme)
		err = k8sClient.Patch(ctx, ns, patchedNs)
		Expect(err).NotTo(HaveOccurred())
	}
}

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
		paas         api.Paas
		paasConfig   api.PaasConfig
		ctx          context.Context
		reconciler   *PaasReconciler
		namespaces   = []string{ns1, ns2}
		paasNsGroups = []string{group1, group2}
	)
	BeforeEach(func() {
		ctx = context.Background()
		paas = api.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
			},
			Spec: api.PaasSpec{
				Requestor: "somebody",
				Capabilities: api.PaasCapabilities{
					enabledCapName:   api.PaasCapability{Enabled: true},
					disabledCapName1: api.PaasCapability{},
				},
				Namespaces: namespaces,
				Groups: api.PaasGroups{
					group1: api.PaasGroup{Query: group1Query},
					group2: api.PaasGroup{Users: []string{"usr2"}},
					group3: api.PaasGroup{Users: []string{"usr3"}},
				},
				Quota: quota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
			},
		}
		assurePaas(ctx, &paas)
		paasConfig = api.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: api.PaasConfigSpec{
				Capabilities: map[string]api.ConfigCapability{
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
			It("should return a nsdef for a ns named after the paas", func() {
				Expect(nsDefs).To(HaveKey(paasName))
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
					"defaultns":      {namespace: paasName},
					ns1:              {namespace: join(paasName, ns1)},
					enabledCapName:   {namespace: join(paasName, enabledCapName)},
					disabledCapName1: {namespace: join(paasName, disabledCapName1)},
					disabledCapName2: {namespace: join(paasName, disabledCapName1)},
					"recursive":      {namespace: join(paasName, "mypns", ns1)},
				}
				for pnsName, pnsDef := range myPnss {
					assureNamespaceWithPaasReference(ctx, pnsDef.namespace, paasName)
					var pns = api.PaasNS{
						ObjectMeta: metav1.ObjectMeta{
							Name:      join("mypns", pnsName),
							Namespace: pnsDef.namespace,
						},
						Spec: api.PaasNSSpec{
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
			It("should return a nsdef for every paasns in default ns", func() {
				Expect(nsDefs).To(HaveKey(join(paasName, "mypns", "defaultns")))
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
					"nongroups": {namespace: paasName},
					"groups":    {namespace: paasName, groups: paasNsGroups},
				}
				for pnsName, pnsDef := range myPnss {
					assureNamespaceWithPaasReference(ctx, pnsDef.namespace, paasName)
					var pns = api.PaasNS{
						ObjectMeta: metav1.ObjectMeta{
							Name:      pnsName,
							Namespace: pnsDef.namespace,
						},
						Spec: api.PaasNSSpec{
							Paas:   paasName,
							Groups: pnsDef.groups,
						},
					}
					err := reconciler.Create(ctx, &pns)
					Expect(err).NotTo(HaveOccurred())
					validatePaasNSExists(ctx, paasName, pnsName)
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
				Expect(withoutGroups).To(ContainElement(join(paasName, group2)))
				Expect(withoutGroups).NotTo(ContainElement(group2))
				Expect(withoutGroups).To(ContainElement(join(paasName, group3)))
				Expect(withoutGroups).NotTo(ContainElement(group3))
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
	})
})

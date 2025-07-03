package controller

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Clusterrolebindings", Ordered, func() {
	const (
		serviceName        = "crb"
		capName            = serviceName
		capAppSetName      = capName + "-as"
		capAppSetNamespace = "asns"
		paasName           = capName + "-test"
		capNSName          = paasName + "-" + capName
		secondPaasName     = capName + "-other"
		secondCapNSName    = secondPaasName + "-" + capName
	)
	var (
		ctx             context.Context
		paas            *v1alpha2.Paas
		paasNsDefs      namespaceDefs
		secondPaas      *v1alpha2.Paas
		secondNsDefs    namespaceDefs
		reconciler      *PaasReconciler
		paasConfig      v1alpha2.PaasConfig
		capConfig       v1alpha2.ConfigCapability
		defaultCapPerms = []string{capName + "-view"}
		extraCapPerms   = []string{capName + "-edit"}
	)
	BeforeAll(func() {
		capConfig = v1alpha2.ConfigCapability{
			AppSet: capAppSetName,
			DefaultPermissions: v1alpha2.ConfigCapPerm{
				capName: defaultCapPerms,
			},
			ExtraPermissions: v1alpha2.ConfigCapPerm{
				capName: extraCapPerms,
			},
		}
		paasConfig = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: v1alpha2.PaasConfigSpec{
				ClusterWideArgoCDNamespace: capAppSetNamespace,
				Capabilities: map[string]v1alpha2.ConfigCapability{
					capName: capConfig,
				},
			},
		}
	})
	BeforeEach(func() {
		ctx = context.Background()
		config.SetConfig(paasConfig)
		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		paas = &v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
				UID:  "abc", // Needed or owner references fail
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: capName,
				Capabilities: v1alpha2.PaasCapabilities{
					capName: v1alpha2.PaasCapability{
						ExtraPermissions: true,
					},
				},
			},
		}
		paasNsDefs = namespaceDefs{
			capNSName: namespaceDef{nsName: capNSName, capName: capName, capConfig: capConfig, quotaName: capNSName},
		}
		secondNsDefs = namespaceDefs{
			secondCapNSName: namespaceDef{
				nsName: secondCapNSName, capName: capName, capConfig: capConfig,
				quotaName: secondCapNSName,
			},
		}
		secondPaas = paas.DeepCopy()
		secondPaas.Name = secondPaasName
	})
	When("managing clusterRoleBindings for a paas with capability with crb config", func() {
		Context("while creating the Paas with extra permissions enabled", Ordered, func() {
			It("should succeed", func() {
				err := reconciler.reconcileClusterRoleBindings(ctx, paas, paasNsDefs)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should create clusterRoleBindings for default permissions", func() {
				for _, crbRole := range defaultCapPerms {
					crbName := join("paas", crbRole)
					var crb rbac.ClusterRoleBinding
					err := reconciler.Get(ctx, types.NamespacedName{Name: crbName}, &crb)
					Expect(err).NotTo(HaveOccurred())
					Expect(crb.RoleRef).To(Equal(
						rbac.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: crbRole}))
					Expect(crb.Subjects).To(ContainElement(
						rbac.Subject{Kind: "ServiceAccount", APIGroup: "", Name: capName, Namespace: capNSName}))
				}
			})
			It("should create clusterRoleBindings for extra permissions", func() {
				for _, crbRole := range extraCapPerms {
					crbName := join("paas", crbRole)
					var crb rbac.ClusterRoleBinding
					err := reconciler.Get(ctx, types.NamespacedName{Name: crbName}, &crb)
					Expect(err).NotTo(HaveOccurred())
					Expect(crb.RoleRef).To(Equal(
						rbac.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: crbRole}))
					Expect(crb.Subjects).To(ContainElement(
						rbac.Subject{Kind: "ServiceAccount", APIGroup: "", Name: capName, Namespace: capNSName}))
				}
			})
		})
		Context("while creating the Paas with extra permissions disabled", Ordered, func() {
			It("should succeed", func() {
				capability := paas.Spec.Capabilities[capName]
				capability.ExtraPermissions = false
				paas.Spec.Capabilities[capName] = capability
				err := reconciler.reconcileClusterRoleBindings(ctx, paas, paasNsDefs)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not create clusterRoleBindings for extra permissions", func() {
				for _, crbRole := range extraCapPerms {
					crbName := join("paas", crbRole)
					var crb rbac.ClusterRoleBinding
					err := reconciler.Get(ctx, types.NamespacedName{Name: crbName}, &crb)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(
						fmt.Sprintf("clusterrolebindings.rbac.authorization.k8s.io \"%s\" not found", crbName)))
				}
			})
		})
		Context("while disabling the capability", Ordered, func() {
			It("should succeed", func() {
				delete(paas.Spec.Capabilities, capName)
				err := reconciler.reconcileClusterRoleBindings(ctx, paas, paasNsDefs)
				Expect(err).NotTo(HaveOccurred())
			})
			It("CRB's for capability should be removed", func() {
				for _, crbRole := range append(defaultCapPerms, extraCapPerms...) {
					crbName := join("paas", crbRole)
					var crb rbac.ClusterRoleBinding
					err := reconciler.Get(ctx, types.NamespacedName{Name: crbName}, &crb)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(
						fmt.Sprintf("clusterrolebindings.rbac.authorization.k8s.io \"%s\" not found", crbName)))
				}
			})
		})
		Context("while adding a second Paas", Ordered, func() {
			It("should succeed", func() {
				var err error
				err = reconciler.reconcileClusterRoleBindings(ctx, paas, paasNsDefs)
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.reconcileClusterRoleBindings(ctx, secondPaas, secondNsDefs)
				Expect(err).NotTo(HaveOccurred())
			})
			It("CRB's for capability should not be removed", func() {
				for _, crbRole := range append(defaultCapPerms, extraCapPerms...) {
					crbName := join("paas", crbRole)
					var crb rbac.ClusterRoleBinding
					err := reconciler.Get(ctx, types.NamespacedName{Name: crbName}, &crb)
					Expect(err).NotTo(HaveOccurred())
					Expect(crb.RoleRef).To(Equal(
						rbac.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: crbRole}))
					for _, nsName := range []string{capNSName, secondCapNSName} {
						Expect(crb.Subjects).To(ContainElement(
							rbac.Subject{Kind: "ServiceAccount", APIGroup: "", Name: capName, Namespace: nsName}))
					}
				}
			})
		})
		Context("while removing the Paas when it is not the last", Ordered, func() {
			It("should succeed", func() {
				var err error
				err = reconciler.reconcileClusterRoleBindings(ctx, paas, paasNsDefs)
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.reconcileClusterRoleBindings(ctx, secondPaas, secondNsDefs)
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.finalizePaasClusterRoleBindings(ctx, paas)
				Expect(err).NotTo(HaveOccurred())
			})
			It("CRB's for capability should not be removed", func() {
				for _, crbRole := range append(defaultCapPerms, extraCapPerms...) {
					crbName := join("paas", crbRole)
					var crb rbac.ClusterRoleBinding
					err := reconciler.Get(ctx, types.NamespacedName{Name: crbName}, &crb)
					Expect(err).NotTo(HaveOccurred())
					Expect(crb.RoleRef).To(Equal(
						rbac.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: crbRole}))
					Expect(crb.Subjects).To(ContainElement(
						rbac.Subject{Kind: "ServiceAccount", APIGroup: "", Name: capName, Namespace: secondCapNSName}))
				}
			})
		})
		Context("while removing the Paas when it is the last", Ordered, func() {
			It("should succeed", func() {
				var err error
				err = reconciler.finalizePaasClusterRoleBindings(ctx, paas)
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.finalizePaasClusterRoleBindings(ctx, secondPaas)
				Expect(err).NotTo(HaveOccurred())
			})
			It("CRB's for capability should be removed", func() {
				for _, crbRole := range append(defaultCapPerms, extraCapPerms...) {
					crbName := join("paas", crbRole)
					var crb rbac.ClusterRoleBinding
					err := reconciler.Get(ctx, types.NamespacedName{Name: crbName}, &crb)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(
						fmt.Sprintf("clusterrolebindings.rbac.authorization.k8s.io \"%s\" not found", crbName)))
				}
			})
		})
	})
})

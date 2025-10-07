package controller

import (
	"context"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/config"
	paasquota "github.com/belastingdienst/opr-paas/v3/pkg/quota"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Rolebinding", Ordered, func() {
	const (
		paasRequestor = "rb-controller"
		capName       = "argocd"
		groupName     = "my-paas-group"
		funcRoleName  = "my-func-role"
		tecRole1      = "admin"
		tecRole2      = "view"
		lbl1Key       = "key1"
		lbl1Value     = "value1"
		lbl2Key       = "key2"
		lbl2Value     = "value2"
		manByLbl      = "manbylbl"
		manBySuffix   = "manby"
		reqLbl        = "requestor-label"
		qtaLbl        = "quota-label"
		kubeInstLabel = "app.kubernetes.io/instance"
	)
	var (
		paas       *v1alpha2.Paas
		reconciler *PaasReconciler
		myConfig   v1alpha2.PaasConfig
		paasName   = paasRequestor
		ns1        = join(paasRequestor, "ns1")
	)
	ctx := context.Background()

	BeforeEach(func() {
		paasName = paasRequestor
		paas = &v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				UID:  "MY-UID",
				Name: paasName,
				Labels: map[string]string{
					lbl1Key:       lbl1Value,
					lbl2Key:       lbl2Value,
					kubeInstLabel: "whatever",
				},
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: paasRequestor,
				Capabilities: v1alpha2.PaasCapabilities{
					capName: v1alpha2.PaasCapability{},
				},
				Quota: paasquota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
				Groups: v1alpha2.PaasGroups{
					groupName: v1alpha2.PaasGroup{Users: []string{"u1", "u2"}, Roles: []string{funcRoleName}},
				},
				Namespaces: v1alpha2.PaasNamespaces{
					ns1: {},
				},
			},
		}
		myConfig = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: v1alpha2.PaasConfigSpec{
				Capabilities: map[string]v1alpha2.ConfigCapability{
					capName: {
						QuotaSettings: v1alpha2.ConfigQuotaSettings{
							DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
								corev1.ResourceLimitsCPU: resourcev1.MustParse("5"),
							},
						},
					},
				},
				ManagedByLabel:  manByLbl,
				ManagedBySuffix: manBySuffix,
				RequestorLabel:  reqLbl,
				QuotaLabel:      qtaLbl,
				RoleMappings: v1alpha2.ConfigRoleMappings{
					funcRoleName: []string{tecRole1, tecRole2},
				},
				Templating: v1alpha2.ConfigTemplatingItems{
					RoleBindingLabels: v1alpha2.ConfigTemplatingItem{
						//revive:disable-next-line
						"": "{{ range $key, $value := .Paas.Labels }}{{ if ne $key \"" + kubeInstLabel + "\" }}{{$key}}: {{$value}}\n{{end}}{{end}}",
					},
				},
			},
		}

		// Updates context to include paasConfig
		ctx = context.WithValue(ctx, config.ContextKeyPaasConfig, myConfig)

		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		assureNamespace(ctx, ns1)
	})

	When("reconciling rolebindings for a namespace", func() {
		expectedTecRoles := []string{tecRole1, tecRole2}

		It("reconciles successfully", func() {
			err := reconciler.reconcileNamespaceRolebindings(ctx, paas, nil, ns1)
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates all rolebindings as expected", func() {
			for _, tecRole := range expectedTecRoles {
				rbName := join("paas", tecRole)
				var rb rbac.RoleBinding
				err := reconciler.Get(ctx, types.NamespacedName{Name: rbName, Namespace: ns1}, &rb)
				Expect(err).NotTo(HaveOccurred())
			}
		})
		It("have set all expected labels", func() {
			var (
				expectedLabels = map[string]string{
					lbl1Key: lbl1Value,
					lbl2Key: lbl2Value,
				}
			)
			var rb rbac.RoleBinding
			err := reconciler.Get(ctx, types.NamespacedName{Namespace: ns1, Name: join("paas", tecRole1)}, &rb)
			Expect(err).NotTo(HaveOccurred())
			for key, value := range expectedLabels {
				Expect(rb.ObjectMeta.Labels).To(HaveKeyWithValue(key, value))
			}
			Expect(rb.ObjectMeta.Labels).NotTo(HaveKey(kubeInstLabel))
		})
	})
})

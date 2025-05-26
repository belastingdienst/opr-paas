/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/fields"
	paasquota "github.com/belastingdienst/opr-paas/internal/quota"
	argocd "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	quotav1 "github.com/openshift/api/quota/v1"
	userv1 "github.com/openshift/api/user/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func getConditionsFromPaas(paas *api.Paas) map[string]metav1.Condition {
	conditions := map[string]metav1.Condition{}
	for _, condition := range paas.Status.Conditions {
		conditions[condition.Type] = condition
	}
	return conditions
}

var _ = Describe("Get paas from ns", func() {
	const (
		paasName = "my-paas"
		nsName   = paasName + "-myns"
	)
	var (
		controller    = true
		notController = false
	)
	When("using proper reference", func() {
		It("should return the name, and nil", func() {
			ns := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Paas", Name: paasName, Controller: &controller},
						{Kind: "SomethingElse", Name: paasName + "2", Controller: &controller},
						{Kind: "Paas", Name: paasName + "3", Controller: &notController},
					},
				},
			}
			name, err := paasFromNs(ns)
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal(paasName))
		})
	})
	When("having no Paas references", func() {
		It("should return empty string and error", func() {
			ns := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "SomethingElse", Name: paasName + "2", Controller: &controller},
						{Kind: "Paas", Name: paasName + "3", Controller: &notController},
					},
				},
			}
			name, err := paasFromNs(ns)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(
				"failed to get owner reference with kind paas and controller=true from namespace")))
			Expect(name).To(BeEmpty())
		})
	})
	When("having multiple Paas references", func() {
		It("should return empty string and error", func() {
			ns := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Paas", Name: paasName, Controller: &controller},
						{Kind: "SomethingElse", Name: paasName + "2", Controller: &controller},
						{Kind: "Paas", Name: paasName + "3", Controller: &notController},
						{Kind: "Paas", Name: paasName + "4", Controller: &controller},
					},
				},
			}
			name, err := paasFromNs(ns)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(
				"found multiple owner references with kind paas and controller=true")))
			Expect(name).To(BeEmpty())
		})
	})
	When("having improper prefix", func() {
		It("should return empty string and error", func() {
			ns := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "other-" + nsName,
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Paas", Name: paasName, Controller: &controller},
						{Kind: "SomethingElse", Name: paasName + "2", Controller: &controller},
						{Kind: "Paas", Name: paasName + "3", Controller: &notController},
					},
				},
			}
			name, err := paasFromNs(ns)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(
				"namespace is not prefixed with paasName in owner reference")))
			Expect(name).To(BeEmpty())
		})
	})
})

var _ = Describe("Paas Controller", Ordered, func() {
	const (
		paasRequestor      = "paas-controller"
		capAppSetNamespace = "asns"
		capAppSetName      = "argoas"
		capName            = "argocd"
		paasSystem         = "paasnssystem"
		paasPkSecret       = "paasns-pk-secret"
		paasWithArgoCDName = paasRequestor + "-with-argocd"
	)
	var (
		paas         *api.Paas
		appSet       *argocd.ApplicationSet
		reconciler   *PaasReconciler
		request      controllerruntime.Request
		myConfig     v1alpha2.PaasConfig
		paasName     = paasRequestor
		capNamespace = paasName + "-" + capName
		privateKey   []byte
		mycrypt      *crypt.Crypt
		paasSecret   string
	)
	ctx := context.Background()

	BeforeAll(func() {
		var err error
		assureNamespace(ctx, paasSystem)
		mycrypt, privateKey, err = newGeneratedCrypt(paasName)
		if err != nil {
			Fail(err.Error())
		}
		createPaasPrivateKeySecret(ctx, paasSystem, paasPkSecret, privateKey)
		paasSecret, err = mycrypt.Encrypt([]byte("paasSecret"))
		Expect(err).NotTo(HaveOccurred())
		assureNamespace(ctx, "gsns")
		appSet = &argocd.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      capAppSetName,
				Namespace: capAppSetNamespace,
			},
			Spec: argocd.ApplicationSetSpec{
				Generators: []argocd.ApplicationSetGenerator{},
			},
		}
	})

	BeforeEach(func() {
		paasName = paasRequestor
		paas = &api.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
			},
			Spec: api.PaasSpec{
				Requestor: paasRequestor,
				Capabilities: api.PaasCapabilities{
					capName: api.PaasCapability{
						Enabled: true,
					},
				},
				Quota: paasquota.Quota{
					"cpu": resourcev1.MustParse("1"),
				},
			},
		}
		myConfig = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: v1alpha2.PaasConfigSpec{
				ClusterWideArgoCDNamespace: capAppSetNamespace,
				Capabilities: map[string]v1alpha2.ConfigCapability{
					capName: {
						AppSet: capAppSetName,
						QuotaSettings: v1alpha2.ConfigQuotaSettings{
							DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
								corev1.ResourceLimitsCPU: resourcev1.MustParse("5"),
							},
						},
					},
				},
				Debug: false,
				DecryptKeysSecret: v1alpha2.NamespacedName{
					Name:      paasPkSecret,
					Namespace: paasSystem,
				},
				ManagedByLabel:  "argocd.argoproj.io/manby",
				ManagedBySuffix: "argocd",
				RequestorLabel:  "o.lbl",
				QuotaLabel:      "q.lbl",
			},
		}
		config.SetConfig(myConfig)
		reconciler = &PaasReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	When("requesting schema from Reconciler", func() {
		It("should return a schema", func() {
			Expect(reconciler.getScheme()).NotTo(BeNil())
			Expect(reconciler.getScheme()).To(Equal(k8sClient.Scheme()))
		})
	})

	// getPaasFromRequest
	When("getting a Paas from a request", func() {
		It("should return nil when paas does not exist", func() {
			var err error
			paasName = paasRequestor + "-request-does-not-exist"
			paas.Name = paasName
			request.Name = paasName
			request.NamespacedName = types.NamespacedName{Name: paasName}
			paas, err = reconciler.getPaasFromRequest(ctx, request)
			// Expect(err).To(HaveOccurred())
			// Expect(err.Error()).To(MatchRegexp(`paas.cpet.belastingdienst.nl .* not found`))
			Expect(err).NotTo(HaveOccurred())
			Expect(paas).To(BeNil())
		})

		It("should return nil when paas is being deleted", func() {
			var gracePeriodSeconds = int64(2)
			paasName = paasRequestor + "-request-being-deleted"
			paas.Name = paasName
			assurePaas(ctx, *paas)
			err := reconciler.Delete(ctx, paas, &client.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
			Expect(err).NotTo(HaveOccurred())
			request.Name = paasName
			request.NamespacedName = types.NamespacedName{Name: paasName}
			paas, err = reconciler.getPaasFromRequest(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(paas).To(BeNil())
		})

		It("should properly get a Paas from the request", func() {
			var err error
			paasName = paasRequestor + "-request"
			paas.Name = paasName
			assurePaas(ctx, *paas)
			request.Name = paasName
			request.NamespacedName = types.NamespacedName{Name: paasName}
			paas, err = reconciler.getPaasFromRequest(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(paas).NotTo(BeNil())
			Expect(paas.Name).To(Equal(paasName))
			Expect(controllerutil.ContainsFinalizer(paas, paasFinalizer)).To(BeTrue())
		})
	})

	When("setting state on a Paas", func() {
		// setFinalizing
		It("can set state to finalizing", func() {
			var err error
			paasName = paasRequestor + "-set-finalizing"
			paas.Name = paasName
			assurePaas(ctx, *paas)
			request.Name = paasName
			request.NamespacedName = types.NamespacedName{Name: paasName}
			paas, err = reconciler.getPaasFromRequest(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			preConditions := getConditionsFromPaas(paas)
			Expect(preConditions).To(HaveKey(api.TypeReadyPaas))
			Expect(preConditions).NotTo(HaveKey(api.TypeDegradedPaas))

			err = reconciler.setFinalizing(ctx, paas)
			Expect(err).NotTo(HaveOccurred())
			paas = getPaas(ctx, paasName)

			postConditions := getConditionsFromPaas(paas)
			Expect(postConditions).To(HaveKey(api.TypeDegradedPaas))
			finalizingCondition := postConditions[api.TypeDegradedPaas]
			Expect(finalizingCondition.Status).To(Equal(metav1.ConditionUnknown))
			Expect(finalizingCondition.Reason).To(Equal("Finalizing"))
			Expect(finalizingCondition.Message).To(Equal(
				fmt.Sprintf("Performing finalizer operations for Paas: %s ", paasName)))
		})
		// setErrorCondition
		It("can set and reset an error and set finalizing state", func() {
			var err error
			paasName = paasRequestor + "-set-error"
			paas.Name = paasName
			assurePaas(ctx, *paas)
			request.Name = paasName
			request.NamespacedName = types.NamespacedName{Name: paasName}
			paas, err = reconciler.getPaasFromRequest(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			preConditions := getConditionsFromPaas(paas)
			Expect(preConditions).To(HaveKey(api.TypeReadyPaas))
			Expect(preConditions).NotTo(HaveKey(api.TypeHasErrorsPaas))
			Expect(preConditions).NotTo(HaveKey(api.TypeDegradedPaas))
			preReadyCondition := preConditions[api.TypeReadyPaas]
			Expect(preReadyCondition.Status).To(Equal(metav1.ConditionUnknown))
			Expect(preReadyCondition.Reason).To(Equal("Reconciling"))
			Expect(preReadyCondition.Message).To(Equal("Starting reconciliation"))

			myError := errors.New("my custom error")
			err = reconciler.setErrorCondition(ctx, paas, myError)
			Expect(err).NotTo(HaveOccurred())
			paas = getPaas(ctx, paasName)

			errorConditions := getConditionsFromPaas(paas)
			Expect(errorConditions).To(HaveKey(api.TypeReadyPaas))
			errorReadyCondition := errorConditions[api.TypeReadyPaas]
			Expect(errorReadyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(errorReadyCondition.Reason).To(Equal("ReconcilingError"))
			Expect(errorReadyCondition.Message).To(Equal(fmt.Sprintf("Reconciling (%s) failed", paasName)))
			Expect(errorConditions).To(HaveKey(api.TypeHasErrorsPaas))
			errorErrorsCondition := errorConditions[api.TypeHasErrorsPaas]
			Expect(errorErrorsCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(errorErrorsCondition.Reason).To(Equal("ReconcilingError"))
			Expect(errorErrorsCondition.Message).To(Equal(myError.Error()))
			Expect(errorConditions).NotTo(HaveKey(api.TypeDegradedPaas))

			err = reconciler.setSuccessfulCondition(ctx, paas)
			Expect(err).NotTo(HaveOccurred())
			paas = getPaas(ctx, paasName)

			resetConditions := getConditionsFromPaas(paas)
			Expect(resetConditions).To(HaveKey(api.TypeReadyPaas))
			resetReadyCondition := resetConditions[api.TypeReadyPaas]
			Expect(resetReadyCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(resetReadyCondition.Reason).To(Equal("Reconciling"))
			Expect(resetReadyCondition.Message).To(Equal(fmt.Sprintf("Reconciled (%s) successfully", paasName)))
			Expect(resetConditions).To(HaveKey(api.TypeHasErrorsPaas))
			resetErrorsCondition := resetConditions[api.TypeHasErrorsPaas]
			Expect(resetErrorsCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(resetErrorsCondition.Reason).To(Equal("Reconciling"))
			Expect(resetErrorsCondition.Message).To(Equal(fmt.Sprintf("Reconciled (%s) successfully", paasName)))
			Expect(resetConditions).NotTo(HaveKey(api.TypeDegradedPaas))

			err = reconciler.setFinalizing(ctx, paas)
			Expect(err).NotTo(HaveOccurred())
			paas = getPaas(ctx, paasName)

			finalizingConditions := getConditionsFromPaas(paas)
			Expect(finalizingConditions).To(HaveKey(api.TypeDegradedPaas))
			finalizingCondition := finalizingConditions[api.TypeDegradedPaas]
			Expect(finalizingCondition.Status).To(Equal(metav1.ConditionUnknown))
			Expect(finalizingCondition.Reason).To(Equal("Finalizing"))
			Expect(finalizingCondition.Message).To(Equal(
				fmt.Sprintf("Performing finalizer operations for Paas: %s ", paasName)))
		})
	})

	When("finalizing a Paas", func() {
		// setFinalizing > see 'setting state' above
		// finalizePaas skipped (only calling sub methods which are tested elsewhere)
		// removeFinalizer
		It("should successfully remove the finalizer", func() {
			var err error
			paasName = paasRequestor + "-remove-finalizer"
			paas.Name = paasName
			assurePaas(ctx, *paas)
			request.Name = paasName
			request.NamespacedName = types.NamespacedName{Name: paasName}
			paas, err = reconciler.getPaasFromRequest(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(paas.Finalizers).To(ContainElement(paasFinalizer))

			err = reconciler.removeFinalizer(ctx, paas)
			Expect(err).NotTo(HaveOccurred())
			paas = getPaas(ctx, paasName)
			Expect(paas.Finalizers).NotTo(ContainElement(paasFinalizer))
		})
	})
	// Reconcile
	When("reconciling a Paas", func() {
		It("should succeed for normal paas", func() {
			var err error
			var result controllerruntime.Result
			request.Name = paasName
			request.NamespacedName = types.NamespacedName{Name: paasName}
			paas.Spec.SSHSecrets = map[string]string{"validSecret": paasSecret}
			result, err = reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(controllerruntime.Result{}))
		})

		// getPaasFromRequest (paas==nil)
		It("should return nil when paas does not exist", func() {
			var err error
			var result controllerruntime.Result
			paasName = paasRequestor + "-non-existent"
			paas.Name = paasName
			// assurePaas(ctx, paas)
			request.Name = paasName
			request.NamespacedName = types.NamespacedName{Name: paasName}
			result, err = reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(controllerruntime.Result{}))
		})

		// paasReconcilers return err (checking failure when cap does not exist)
		It("should return error when a paasReconciler method returns an error", func() {
			var err error
			var result controllerruntime.Result
			paasName = paasRequestor + "-non-existent-cap"
			brokenPaas := paas.DeepCopy()
			brokenPaas.Name = paasName
			brokenPaas.Spec.Capabilities["non-existent"] = api.PaasCapability{
				Enabled: true,
			}
			assurePaas(ctx, *brokenPaas)
			request.Name = paasName
			request.NamespacedName = types.NamespacedName{Name: paasName}
			result, err = reconciler.Reconcile(ctx, request)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("a capability is requested, but not configured")))
			Expect(result).To(Equal(controllerruntime.Result{}))
		})

		// error from nsDefsFromPaas not unittested.
		// Only occurs if cap does not exist, which is prevented by webhook, and a paasReconciler error raises first
		// It("should return error when nsDefsFromPaas method returns an error", func() {
		// })

		// paasNsReconcilers returns error
		It("should return error when a paasNsReconciler method returns an error", func() {
			var err error
			var result controllerruntime.Result
			paasName = paasRequestor + "-secret-failure"
			brokenPaas := paas.DeepCopy()
			brokenPaas.Name = paasName
			brokenPaas.Spec.SSHSecrets = map[string]string{"broken": paasSecret}
			assurePaas(ctx, *brokenPaas)
			request.Name = paasName
			request.NamespacedName = types.NamespacedName{Name: paasName}
			assureAppSet(ctx, capAppSetName, capAppSetNamespace)
			result, err = reconciler.Reconcile(ctx, request)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to decrypt secret")))
			Expect(result).To(Equal(controllerruntime.Result{}))
		})

		// ensureAppSetCaps returns error is very difficult to test on it's own. Skipping.
	})

	When("reconciling a Paas with argocd capability", func() {
		It("should not return an error", func() {
			paas.Name = paasWithArgoCDName
			request.Name = paasWithArgoCDName
			capNamespace = paasWithArgoCDName + "-" + capName
			assurePaas(ctx, *paas)
			assureNamespace(ctx, capNamespace)
			patchAppSet(ctx, appSet)
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter.Microseconds()).To(BeZero())
		})
		It("should create an appset entry", func() {
			a := &argocd.ApplicationSet{}
			appSetName := types.NamespacedName{
				Name:      capAppSetName,
				Namespace: capAppSetNamespace,
			}
			err := k8sClient.Get(ctx, appSetName, a)
			Expect(err).NotTo(HaveOccurred())
			entries := make(fields.Entries)
			for _, generator := range a.Spec.Generators {
				generatorEntries, err := fields.EntriesFromJSON(generator.List.Elements)
				Expect(err).NotTo(HaveOccurred())
				entries = entries.Merge(generatorEntries)
			}
			Expect(entries).To(HaveKey(paasWithArgoCDName))
		})
	})

	When("reconciling a Paas without argocd capability", func() {
		It("should not return an error", func() {
			paasName = paasRequestor + "-nocap"
			paas.Name = paasName
			paas.Spec.Capabilities = make(api.PaasCapabilities)
			request.Name = paasName
			capNamespace = paasName + "-" + capName
			assurePaas(ctx, *paas)
			assureNamespace(ctx, capNamespace)
			patchAppSet(ctx, appSet)
			paas.Spec.Capabilities = make(api.PaasCapabilities)
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter.Microseconds()).To(BeZero())
		})

		It("should not create an appset entry", func() {
			appSet := &argocd.ApplicationSet{}
			appSetName := types.NamespacedName{
				Name:      capAppSetName,
				Namespace: capAppSetNamespace,
			}
			err := k8sClient.Get(ctx, appSetName, appSet)
			Expect(err).NotTo(HaveOccurred())
			entries := make(fields.Entries)
			for _, generator := range appSet.Spec.Generators {
				generatorEntries, err := fields.EntriesFromJSON(generator.List.Elements)
				Expect(err).NotTo(HaveOccurred())
				entries = entries.Merge(generatorEntries)
			}
			Expect(entries).NotTo(HaveKey(paasName))
		})
	})
})

var _ = Describe("Paas Reconcile", Ordered, func() {
	const (
		paasName           = "paas-reconcile"
		capAppSetNamespace = paasName + "-asns"
		capAppSetName      = "argoas"
		capName            = "recon"
		capNamespace       = paasName + "-" + capName
		paasSystem         = "recon-nssystem"
		paasPkSecret       = "recon-secret"
		nsName             = "myns"
		paasNSName         = "mypaasns"
		groupName          = "prcn-mygroup"
		ldapGroupName      = "prcn-myldapgroup"
		ldapGroupQuery     = "CN=" + ldapGroupName + ",OU=org_unit,DC=example,DC=org"
		funcRoleName1      = "myfuncrole1"
		funcRoleName2      = "myfuncrole2"
		techRoleName1      = "mytechrole1"
		techRoleName2      = "mytechrole2"
		defaultPermSA      = "def-perm-service-account"
		defaultPermCR      = "def-parm-cluster-role"
		extraPermSA        = "extra-perm-service-account"
		extraPermCR        = "extra-parm-cluster-role"
	)
	var (
		paas                 *api.Paas
		reconciler           *PaasReconciler
		request              controllerruntime.Request
		myConfig             v1alpha2.PaasConfig
		privateKey           []byte
		mycrypt              *crypt.Crypt
		secretValue          string
		secretEncryptedValue string
		secretName           = "my-secret"
		secretHashedName     = fmt.Sprintf("paas-ssh-%s", strings.ToLower(hashData(secretName)[:8]))
		userGroupName        = join(paasName, groupName)
		rolebindings         = []string{techRoleName1, techRoleName2}
		clusterRolebindings  = map[string][]string{
			defaultPermSA: {defaultPermCR}, extraPermSA: {extraPermCR}}
	)
	ctx := context.Background()
	BeforeAll(func() {
		var err error
		assureNamespace(ctx, paasSystem)
		mycrypt, privateKey, err = newGeneratedCrypt(paasName)
		if err != nil {
			Fail(err.Error())
		}
		createPaasPrivateKeySecret(ctx, paasSystem, paasPkSecret, privateKey)
		secretEncryptedValue, err = mycrypt.Encrypt([]byte(secretValue))
		Expect(err).NotTo(HaveOccurred())
		assureNamespace(ctx, capAppSetNamespace)
		assureAppSet(ctx, capAppSetName, capAppSetNamespace)
		paas = &api.Paas{
			// We need to set this for AmIOwner to work properly
			TypeMeta: metav1.TypeMeta{
				Kind:       "Paas",
				APIVersion: api.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
			},
			Spec: api.PaasSpec{
				Requestor: paasName,
				Capabilities: api.PaasCapabilities{
					capName: api.PaasCapability{
						Enabled:          true,
						ExtraPermissions: true,
					},
				},
				Quota:      paasquota.Quota{"cpu": resourcev1.MustParse("1")},
				Namespaces: []string{nsName},
				Groups: api.PaasGroups{
					groupName:     api.PaasGroup{Roles: []string{funcRoleName1}},
					ldapGroupName: api.PaasGroup{Roles: []string{funcRoleName2}, Query: ldapGroupQuery},
				},
				SSHSecrets: map[string]string{secretName: secretEncryptedValue},
			},
		}
		Expect(paas.Kind).To(Equal("Paas"))
		request.Name = paasName
		assurePaas(ctx, *paas)
		Expect(paas.Kind).To(Equal("Paas"))
		myConfig = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas-config",
			},
			Spec: v1alpha2.PaasConfigSpec{
				ClusterWideArgoCDNamespace: capAppSetNamespace,
				Capabilities: map[string]v1alpha2.ConfigCapability{
					capName: {
						AppSet: capAppSetName,
						QuotaSettings: v1alpha2.ConfigQuotaSettings{
							DefQuota: map[corev1.ResourceName]resourcev1.Quantity{
								corev1.ResourceLimitsCPU: resourcev1.MustParse("5"),
							},
						},
						DefaultPermissions: v1alpha2.ConfigCapPerm{defaultPermSA: []string{defaultPermCR}},
						ExtraPermissions:   v1alpha2.ConfigCapPerm{extraPermSA: []string{extraPermCR}},
					},
				},
				DecryptKeysSecret: v1alpha2.NamespacedName{Name: paasPkSecret, Namespace: paasSystem},
				ManagedByLabel:    "argocd.argoproj.io/manby",
				ManagedBySuffix:   "argocd",
				RequestorLabel:    "o.lbl",
				QuotaLabel:        "q.lbl",
				RoleMappings: v1alpha2.ConfigRoleMappings{
					funcRoleName1: []string{techRoleName1},
					funcRoleName2: []string{techRoleName2},
				},
			},
		}
		config.SetConfig(myConfig)
		reconciler = &PaasReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
	})
	// create Paas
	When("creating a Paas and PaasNS", func() {
		namespaces := []string{join(paasName, nsName), join(paasName, capName), join(paasName, paasNSName)}
		It("should reconcile successfully", func() {
			assurePaas(ctx, *paas)
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			assurePaasNS(ctx,
				api.PaasNS{
					ObjectMeta: metav1.ObjectMeta{Name: paasNSName, Namespace: join(paasName, nsName)},
					Spec: api.PaasNSSpec{
						Paas: paasName},
				})
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(controllerruntime.Result{}))
		})
		It("should have created paas quotas", func() {
			var quotas = []string{paasName, capNamespace}
			for _, quotaName := range quotas {
				var quota quotav1.ClusterResourceQuota
				err := reconciler.Get(ctx, types.NamespacedName{Name: quotaName}, &quota)
				Expect(err).ToNot(HaveOccurred())
			}
		})
		It("should have created paas user groups", func() {
			var group userv1.Group
			err := reconciler.Get(ctx, types.NamespacedName{Name: userGroupName}, &group)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should not have created paas ldap groups", func() {
			var group userv1.Group
			err := reconciler.Get(ctx, types.NamespacedName{Name: ldapGroupName}, &group)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("groups.user.openshift.io \"" + ldapGroupName + "\" not found"))
		})
		It("should have created paas namespaces", func() {
			for _, nsName := range namespaces {
				var ns corev1.Namespace
				err := reconciler.Get(ctx, types.NamespacedName{Name: nsName}, &ns)
				Expect(err).ToNot(HaveOccurred())
			}
		})
		It("should have created paas clusterrolebindings", func() {
			for crbSAName, crbRoleNames := range clusterRolebindings {
				for _, crbRoleName := range crbRoleNames {
					var crb rbac.ClusterRoleBinding
					err := reconciler.Get(ctx, types.NamespacedName{Name: join("paas", crbRoleName)}, &crb)
					Expect(err).ToNot(HaveOccurred())
					Expect(crb.Subjects).To(ContainElement(
						rbac.Subject{
							Kind:      "ServiceAccount",
							APIGroup:  "",
							Name:      crbSAName,
							Namespace: capNamespace,
						},
					))
				}
			}
		})
		It("should have created paas appset list generator entries", func() {
			var capAppSet argocd.ApplicationSet
			err := reconciler.Get(ctx,
				types.NamespacedName{Namespace: capAppSetNamespace, Name: capAppSetName}, &capAppSet)
			Expect(err).ToNot(HaveOccurred())
			Expect(capAppSet.Spec.Generators).To(HaveLen(1))
			list := getListGen(capAppSet.Spec.Generators)
			Expect(list).NotTo(BeNil())
			entries, err := fields.EntriesFromJSON(list.List.Elements)
			Expect(err).ToNot(HaveOccurred())
			Expect(entries).To(HaveLen(1))
			Expect(entries).To(HaveKey(paasName))
		})
		It("should have created paas rolebindings", func() {
			for _, nsName := range namespaces {
				fmt.Fprintf(GinkgoWriter, "DEBUG - Namespace: %v", nsName)
				for _, rbName := range rolebindings {
					var rb rbac.RoleBinding
					err := reconciler.Get(ctx,
						types.NamespacedName{Namespace: nsName, Name: join("paas", rbName)}, &rb)
					Expect(err).ToNot(HaveOccurred())
				}
			}
		})
		It("should have created paas secrets", func() {
			for _, nsName := range namespaces {
				var secret corev1.Secret
				err := reconciler.Get(ctx, types.NamespacedName{Namespace: nsName, Name: secretHashedName}, &secret)
				Expect(err).ToNot(HaveOccurred())
			}
		})
	})
	When("modifying a Paas", Ordered, func() {
		It("should reconcile successfully", func() {
			assurePaas(ctx, *paas)
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			assurePaasNS(ctx,
				api.PaasNS{ObjectMeta: metav1.ObjectMeta{Name: paasNSName, Namespace: join(paasName, nsName)},
					Spec: api.PaasNSSpec{
						Paas: paasName},
				})
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(controllerruntime.Result{}))

			patch := client.MergeFrom(paas.DeepCopy())
			paas.Spec.Namespaces = nil
			paas.Spec.Groups = nil
			paas.Spec.Capabilities = nil
			paas.Spec.SSHSecrets = nil
			err = reconciler.Patch(ctx, paas, patch)
			Expect(err).NotTo(HaveOccurred())
			patchedPaas := getPaas(ctx, paasName)
			Expect(patchedPaas.Spec.Namespaces).To(BeEmpty())
			Expect(patchedPaas.Spec.Groups).To(BeEmpty())
			Expect(patchedPaas.Spec.Capabilities).To(BeEmpty())
			Expect(patchedPaas.Spec.SSHSecrets).To(BeEmpty())
			_, err = reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
		})
		It("should have deleted paas quotas for removed capability", func() {
			var quota quotav1.ClusterResourceQuota
			err := reconciler.Get(ctx, types.NamespacedName{Name: capNamespace}, &quota)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				"clusterresourcequotas.quota.openshift.io \"" + capNamespace + "\" not found"))
		})
		It("should successfully remove user groups", func() {
			var group userv1.Group
			err := reconciler.Get(ctx, types.NamespacedName{Name: groupName}, &group)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("groups.user.openshift.io \"" + groupName + "\" not found"))
		})
		It("should successfully finalize disabled capabilities", func() {
			var capAppSet argocd.ApplicationSet
			err := reconciler.Get(ctx,
				types.NamespacedName{Namespace: capAppSetNamespace, Name: capAppSetName}, &capAppSet)
			Expect(err).ToNot(HaveOccurred())
			Expect(capAppSet.Spec.Generators).To(HaveLen(1))
			list := getListGen(capAppSet.Spec.Generators)
			Expect(list).To(BeNil())
		})
		It("should successfully finalize removed namespaces", func() {
			deletedNamespaces := []string{join(paasName, nsName), join(paasName, capName), join(paasName, paasNSName)}
			for _, nsName := range deletedNamespaces {
				fmt.Fprintf(GinkgoWriter, "DEBUG - Namespace: %v", nsName)
				var ns corev1.Namespace
				err := reconciler.Get(ctx, types.NamespacedName{Name: nsName}, &ns)
				Expect(err).NotTo(HaveOccurred())
				Expect(ns.DeletionTimestamp).NotTo(BeNil())
			}
		})
		It("should have removed paas clusterrolebindings", func() {
			for _, crbRoleNames := range clusterRolebindings {
				for _, crbRoleName := range crbRoleNames {
					var crb rbac.ClusterRoleBinding
					err := reconciler.Get(ctx, types.NamespacedName{Name: join("paas", crbRoleName)}, &crb)
					Expect(err).To(HaveOccurred())
				}
			}
		})
		It("should have removed paas appset list generator entries", func() {
			var capAppSet argocd.ApplicationSet
			err := reconciler.Get(ctx,
				types.NamespacedName{Namespace: capAppSetNamespace, Name: capAppSetName}, &capAppSet)
			Expect(err).ToNot(HaveOccurred())
			Expect(capAppSet.Spec.Generators).To(HaveLen(1))
			list := getListGen(capAppSet.Spec.Generators)
			Expect(list).To(BeNil())
		})
	})
	When("finalizing a Paas", Ordered, func() {
		It("should finalize successfully", func() {
			assurePaas(ctx, *paas)
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			err = reconciler.finalizePaas(ctx, paas)
			Expect(err).NotTo(HaveOccurred())
		})
		It("should have deleted user groups", func() {
			var group userv1.Group
			err := reconciler.Get(ctx, types.NamespacedName{Name: groupName}, &group)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("groups.user.openshift.io \"" + groupName + "\" not found"))
		})
		It("should have deleted paas clusterrolebindings", func() {
			for _, crbRoleNames := range clusterRolebindings {
				for _, crbRoleName := range crbRoleNames {
					var crb rbac.ClusterRoleBinding
					err := reconciler.Get(ctx, types.NamespacedName{Name: join("paas", crbRoleName)}, &crb)
					Expect(err).To(HaveOccurred())
				}
			}
		})
		It("should have deleted paas appset list generator entries", func() {
			var capAppSet argocd.ApplicationSet
			err := reconciler.Get(ctx,
				types.NamespacedName{Namespace: capAppSetNamespace, Name: capAppSetName}, &capAppSet)
			Expect(err).ToNot(HaveOccurred())
			Expect(capAppSet.Spec.Generators).To(HaveLen(1))
			list := getListGen(capAppSet.Spec.Generators)
			Expect(list).To(BeNil())
		})
	})
})

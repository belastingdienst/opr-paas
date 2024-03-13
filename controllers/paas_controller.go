/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const paasFinalizer = "paas.cpet.belastingdienst.nl/finalizer"

// PaasReconciler reconciles a Paas object
type PaasReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (pr PaasReconciler) GetScheme() *runtime.Scheme {
	return pr.Scheme
}

type Reconciler interface {
	Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error
	Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
	GetScheme() *runtime.Scheme
	Delete(context.Context, client.Object, ...client.DeleteOption) error
}

//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas/finalizers,verbs=update

//+kubebuilder:rbac:groups=quota.openshift.io,resources=clusterresourcequotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=user.openshift.io,resources=groups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=argoproj.io,resources=argocds;applicationsets;applications;appprojects,verbs=create;delete;list;watch;update
//+kubebuilder:rbac:groups=core,resources=secrets;configmaps;namespaces,verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings;clusterrolebindings,verbs=create;delete;get;list;patch;update;watch;escallate
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=bind,resourceNames=admin

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Paas object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *PaasReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	paas := &v1alpha1.Paas{}
	logger := getLogger(ctx, paas, "PaaS", req.Name)
	logger.Info("Reconciling the PAAS object")

	err := r.Get(ctx, req.NamespacedName, paas)
	if err != nil {
		if errors.IsNotFound(err) {
			// Something fishy is going on
			// Maybe someone cleaned the finalizers and then removed the PaaS project?
			logger.Info(req.NamespacedName.Name + " is already gone")
			//return ctrl.Result{}, fmt.Errorf("PaaS object %s already gone", req.NamespacedName)
		}
		return ctrl.Result{}, nil
	} else if paas.GetDeletionTimestamp() != nil {
		logger.Info("PAAS object marked for deletion")
		if controllerutil.ContainsFinalizer(paas, paasFinalizer) {
			logger.Info("Finalizing PaaS")
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizePaaS(ctx, paas); err != nil {
				return ctrl.Result{}, err
			}

			logger.Info("Removing finalizer")
			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(paas, paasFinalizer)
			if err := r.Update(ctx, paas); err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("Finalization finished")
		}
		return ctrl.Result{}, nil
	}

	paas.Status.Truncate()
	defer func() {
		logger.Info("Updating PaaS status", "messages", len(paas.Status.Messages), "quotas", paas.Status.Quota)
		if err = r.Status().Update(ctx, paas); err != nil {
			logger.Error(err, "Updating PaaS status failed")
		}
	}()

	// Add finalizer for this CR
	logger.Info("Adding finalizer for PaaS object")
	if !controllerutil.ContainsFinalizer(paas, paasFinalizer) {
		logger.Info("PaaS object has no finalizer yet")
		controllerutil.AddFinalizer(paas, paasFinalizer)
		logger.Info("Added finalizer for PaaS object")
		if err := r.Update(ctx, paas); err != nil {
			logger.Info("Error updating PaaS object")
			logger.Info(fmt.Sprintf("%v", paas))
			return ctrl.Result{}, err
		}
		logger.Info("Updated PaaS object")
	}

	logger.Info("Creating quotas for PAAS object ")
	// Create quotas if needed
	for _, q := range r.BackendEnabledQuotas(ctx, paas) {
		logger.Info("Creating quota " + q.Name + " for PAAS object ")
		if err := r.EnsureQuota(ctx, paas, req, q); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating quota %s", q.ObjectMeta.Name))
			return ctrl.Result{}, err
		}
	}
	paas.Status.Quota = r.BackendEnabledQuotaStatus(paas)

	for _, name := range r.BackendDisabledQuotas(ctx, paas) {
		logger.Info("Cleaning quota " + name + " for PAAS object ")
		if err := r.FinalizeClusterQuota(ctx, paas, name); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating quota %s", name))
			return ctrl.Result{}, err
		}
	}

	logger.Info("Creating default namespace to hold PaasNs resources for PAAS object")
	if ns, err := BackendNamespace(ctx, paas, paas.Name, paas.Name, r.Scheme); err != nil {
		logger.Error(err, fmt.Sprintf("Failure while defining namespace %s", paas.Name))
		return ctrl.Result{}, err
	} else if err = EnsureNamespace(r.Client, ctx, paas.Status.AddMessage, paas, req, ns, r.Scheme); err != nil {
		logger.Error(err, fmt.Sprintf("Failure while creating namespace %s", paas.Name))
		return ctrl.Result{}, err
	} else {
		logger.Info("Creating PaasNs resources for PAAS object")
		for nsName := range paas.AllEnabledNamespaces() {
			pns := r.GetPaasNs(ctx, paas, nsName, paas.Spec.Groups.Names(), paas.GetNsSshSecrets(nsName))
			if err = r.EnsurePaasNs(ctx, paas, req, pns); err != nil {
				logger.Error(err, fmt.Sprintf("Failure while creating PaasNs %s",
					types.NamespacedName{Name: pns.Name, Namespace: pns.Namespace}))
				return ctrl.Result{}, err
			}
		}
	}

	for _, paasns := range r.pnsFromNs(ctx, paas.ObjectMeta.Name) {
		roles := make(map[string][]string)
		for _, roleList := range getConfig().RoleMappings {
			for _, role := range roleList {
				roles[role] = []string{}
			}
		}
		logger.Info("All roles", "Rolebindings map", roles)
		for groupName, groupRoles := range paas.Spec.Groups.Filtered(paasns.Spec.Groups).Roles() {
			for _, mappedRole := range getConfig().RoleMappings.Roles(groupRoles) {
				if role, exists := roles[mappedRole]; exists {
					roles[mappedRole] = append(role, groupName)
				} else {
					roles[mappedRole] = []string{groupName}
				}
			}
		}
		logger.Info("Creating paas RoleBindings for PAASNS object", "Rolebindings map", roles)
		for roleName, groupKeys := range roles {
			statusMessages := v1alpha1.PaasNsStatus{}
			rbName := types.NamespacedName{Namespace: paasns.NamespaceName(), Name: fmt.Sprintf("paas-%s", roleName)}
			logger.Info("Creating Rolebinding", "role", roleName, "groups", groupKeys)
			rb := backendRoleBinding(ctx, r, paas, rbName, roleName, groupKeys)
			if err := EnsureRoleBinding(ctx, r, &paasns, &statusMessages, rb); err != nil {
				err = fmt.Errorf("failure while creating/updating rolebinding %s/%s: %s", rb.ObjectMeta.Namespace, rb.ObjectMeta.Name, err.Error())
				paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, rb, err.Error())
				return ctrl.Result{}, err
			}
			paas.Status.AddMessages(statusMessages.GetMessages())
		}
	}

	logger.Info("Cleaning obsolete namespaces ")
	if err := r.FinalizePaasNss(ctx, paas); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Creating Argo Project")
	if err := r.EnsureAppProject(ctx, paas); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Creating groups for PAAS object ")
	for _, group := range r.BackendGroups(ctx, paas) {
		if err := r.EnsureGroup(ctx, paas, group); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating group %s", group.ObjectMeta.Name))
			return ctrl.Result{}, err
		}
	}

	logger.Info("Creating ldap groups for PAAS object ")
	if err := r.EnsureLdapGroups(ctx, paas); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Updating PaaS object status")
	paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusReconcile, paas, "succeeded")
	logger.Info("PAAS object succesfully reconciled")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PaasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Paas{}).
		WithEventFilter(
			predicate.Or(
				// Spec updated
				predicate.GenerationChangedPredicate{},
				// Labels updated
				predicate.LabelChangedPredicate{},
			)).
		Complete(r)
}

func (r *PaasReconciler) finalizePaaS(ctx context.Context, paas *v1alpha1.Paas) error {
	logger := getLogger(ctx, paas, "PaaS", "finalizer code")
	logger.Info("Inside PaaS finalizer")
	if err := r.FinalizeAppSetCaps(ctx, paas); err != nil {
		logger.Error(err, "AppSet finalizer error")
		return err
	} else if err = r.FinalizeClusterQuotas(ctx, paas); err != nil {
		logger.Error(err, "Quota finalizer error")
		return err
	} else if cleanedLdapQueries, err := r.FinalizeGroups(ctx, paas); err != nil {
		// The whole idea is that groups (which are resources)
		// can also be ldapGroups (lines in a field in a configmap)
		// ldapGroups are only cleaned if the corresponding group is also cleaned
		logger.Error(err, "Group finalizer error")
		if ldapErr := r.FinalizeLdapGroups(ctx, paas, cleanedLdapQueries); ldapErr != nil {
			logger.Error(ldapErr, "And ldapGroup finalizer error")
		}
		return err
	} else if err = r.FinalizeLdapGroups(ctx, paas, cleanedLdapQueries); err != nil {
		logger.Error(err, "LdapGroup finalizer error")
		return err
	} else if err = r.FinalizeExtraClusterRoleBindings(ctx, paas); err != nil {
		logger.Error(err, "Extra ClusterRoleBindings finalizer error")
		return err
	}
	logger.Info("PaaS succesfully finalized")
	return nil
}

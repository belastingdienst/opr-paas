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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (r *PaasReconciler) GetPaas(
	ctx context.Context,
	req ctrl.Request,
) (paas *v1alpha1.Paas, err error) {
	paas = &v1alpha1.Paas{ObjectMeta: metav1.ObjectMeta{Name: req.Name}}
	logger := getLogger(ctx, paas, "PaaS", req.Name)
	err = r.Get(ctx, req.NamespacedName, paas)
	if err != nil {
		if errors.IsNotFound(err) {
			// Something fishy is going on
			// Maybe someone cleaned the finalizers and then removed the PaaS project?
			logger.Info(req.NamespacedName.Name + " is already gone")
			return nil, nil
			//return ctrl.Result{}, fmt.Errorf("PaaS object %s already gone", req.NamespacedName)
		}
		return nil, err
	} else if paas.GetDeletionTimestamp() != nil {
		logger.Info("PAAS object marked for deletion")
		if controllerutil.ContainsFinalizer(paas, paasFinalizer) {
			logger.Info("Finalizing PaaS")
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizePaaS(ctx, paas); err != nil {
				return nil, err
			}

			logger.Info("Removing finalizer")
			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(paas, paasFinalizer)
			if err := r.Update(ctx, paas); err != nil {
				return nil, err
			}
			logger.Info("Finalization finished")
		}
		return nil, nil
	}

	// Add finalizer for this CR
	logger.Info("Adding finalizer for PaaS object")
	if !controllerutil.ContainsFinalizer(paas, paasFinalizer) {
		logger.Info("PaaS object has no finalizer yet")
		controllerutil.AddFinalizer(paas, paasFinalizer)
		logger.Info("Added finalizer for PaaS object")
		if err := r.Update(ctx, paas); err != nil {
			logger.Info("Error updating PaaS object")
			logger.Info(fmt.Sprintf("%v", paas))
			return nil, err
		}
		logger.Info("Updated PaaS object")
	}

	return paas, nil
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *PaasReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	paas := &v1alpha1.Paas{ObjectMeta: metav1.ObjectMeta{Name: req.Name}}
	logger := getLogger(ctx, paas, "PaaS", req.Name)
	logger.Info("Reconciling the PAAS object")

	if paas, err = r.GetPaas(ctx, req); err != nil {
		logger.Error(err, "could not get PaaS from k8s")
		return
	} else if paas == nil {
		logger.Error(err, "nothing to do")
		return
	}

	paas.Status.Truncate()
	defer func() {
		logger.Info("Updating PaaS status", "messages", len(paas.Status.Messages), "quotas", paas.Status.Quota)
		if err = r.Status().Update(ctx, paas); err != nil {
			logger.Error(err, "Updating PaaS status failed")
		}
	}()

	if err := r.ReconcileQuotas(ctx, paas, logger); err != nil {
		return ctrl.Result{}, err
	} else if err := r.ReconcilePaasNss(ctx, paas, logger); err != nil {
		return ctrl.Result{}, err
	} else if err := r.ReconcileRolebindings(ctx, paas, logger); err != nil {
		return ctrl.Result{}, err
	} else if err := r.EnsureAppProject(ctx, paas, logger); err != nil {
		return ctrl.Result{}, err
	} else if err := r.ReconcileGroups(ctx, paas, logger); err != nil {
		return ctrl.Result{}, err
	} else if err := r.EnsureLdapGroups(ctx, paas); err != nil {
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

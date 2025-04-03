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
	"reflect"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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

//revive:disable:line-length-limit
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas/finalizers,verbs=update

// +kubebuilder:rbac:groups=quota.openshift.io,resources=clusterresourcequotas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=user.openshift.io,resources=groups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=core,resources=secrets;configmaps;namespaces,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings;clusterrolebindings,verbs=create;delete;get;list;patch;update;watch
// It is advised to reduce the scope of this permission by stating the resourceNames of the roles you would like Paas to bind to, in your deployment role.yaml
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=bind
//revive:enable:line-length-limit

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Paas object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//

func (pr *PaasReconciler) GetPaas(
	ctx context.Context,
	req ctrl.Request,
) (paas *v1alpha1.Paas, err error) {
	paas = &v1alpha1.Paas{}
	ctx, logger := logging.GetLogComponent(ctx, "paas")
	if err = pr.Get(ctx, req.NamespacedName, paas); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	// Started reconciling, reset status
	meta.SetStatusCondition(
		&paas.Status.Conditions,
		metav1.Condition{
			Type:               v1alpha1.TypeReadyPaas,
			Status:             metav1.ConditionUnknown,
			ObservedGeneration: paas.Generation,
			Reason:             "Reconciling",
			Message:            "Starting reconciliation",
		},
	)
	meta.RemoveStatusCondition(&paas.Status.Conditions, v1alpha1.TypeHasErrorsPaas)
	if err = pr.Status().Update(ctx, paas); err != nil {
		logger.Err(err).Msg("failed to update Paas status")
		return nil, err
	}

	if err := pr.Get(ctx, req.NamespacedName, paas); err != nil {
		logger.Err(err).Msg("failed to re-fetch Paas")
		return nil, err
	}

	// TODO(portly-halicore-76) Move to admission webhook once available
	// check if Config is set, as reconciling and finalizing without config, leaves object in limbo.
	// this is only an issue when object is being removed, finalizers will not be removed
	// causing the object to be in limbo.
	if reflect.DeepEqual(v1alpha1.PaasConfigSpec{}, config.GetConfig().Spec) {
		logger.Error().Msg(noConfigFoundMsg)
		err = pr.setErrorCondition(
			ctx,
			paas,
			fmt.Errorf(
				//revive:disable-next-line
				"please reach out to your system administrator as there is no Paasconfig available to reconcile against",
			),
		)
		if err != nil {
			logger.Err(err).Msg("failed to update Paas status")
			return nil, err
		}
		return nil, errors.New(noConfigFoundMsg)
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(paas, paasFinalizer) {
		logger.Info().Msg("paas has no finalizer yet")
		if ok := controllerutil.AddFinalizer(paas, paasFinalizer); !ok {
			logger.Err(err).Msg("failed to add finalizer")
			return nil, fmt.Errorf("failed to add finalizer")
		}
		if err := pr.Update(ctx, paas); err != nil {
			logger.Err(err).Msg("error updating Paas")
			return nil, err
		}
		logger.Info().Msg("added finalizer to Paas")
	}

	if paas.GetDeletionTimestamp() != nil {
		logger.Info().Msg("paas marked for deletion")
		return nil, pr.updateFinalizer(ctx, paas)
	}

	return paas, nil
}

func (pr *PaasReconciler) updateFinalizer(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	logger := log.Ctx(ctx)

	if controllerutil.ContainsFinalizer(paas, paasFinalizer) {
		logger.Info().Msg("finalizing Paas")
		// Let's add here a status "Downgrade" to reflect that this resource began its process to be terminated.
		meta.SetStatusCondition(&paas.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.TypeDegradedPaas,
			Status: metav1.ConditionUnknown, Reason: "Finalizing", ObservedGeneration: paas.Generation,
			Message: fmt.Sprintf("Performing finalizer operations for Paas: %s ", paas.Name),
		})

		if err := pr.Status().Update(ctx, paas); err != nil {
			logger.Err(err).Msg("Failed to update Paas status")
			return err
		}
		// Run finalization logic for paasFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := pr.finalizePaas(ctx, paas); err != nil {
			return err
		}

		meta.SetStatusCondition(&paas.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.TypeDegradedPaas,
			Status: metav1.ConditionTrue, Reason: "Finalizing", ObservedGeneration: paas.Generation,
			Message: fmt.Sprintf("Finalizer operations for Paas %s name were successfully accomplished", paas.Name),
		})

		if err := pr.Status().Update(ctx, paas); err != nil {
			logger.Err(err).Msg("Failed to update Paas status")
			return err
		}

		logger.Info().Msg("removing finalizer")
		// Remove paasFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		controllerutil.RemoveFinalizer(paas, paasFinalizer)
		if err := pr.Update(ctx, paas); err != nil {
			return err
		}
		logger.Info().Msg("finalization finished")
	}

	return nil
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (pr *PaasReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	paas := &v1alpha1.Paas{ObjectMeta: metav1.ObjectMeta{Name: req.Name}}
	ctx, logger := logging.SetControllerLogger(ctx, paas, pr.Scheme, req)

	if paas, err = pr.GetPaas(ctx, req); err != nil {
		// TODO(portly-halicore-76) move to admission webhook once available
		// Don't requeue that often when no config is found
		if strings.Contains(err.Error(), noConfigFoundMsg) {
			return ctrl.Result{RequeueAfter: requeueTimeout}, nil
		}
		logger.Err(err).Msg("could not get Paas from k8s")
		return ctrl.Result{}, err
	}

	if paas == nil {
		// r.GetPaas handled all logic and returned a nil object
		return ctrl.Result{}, nil
	}

	paas.Status.Truncate()

	reconcilers := []func(context.Context, *v1alpha1.Paas) error{
		pr.ReconcileQuotas,
		pr.ReconcileClusterWideQuota,
		pr.ReconcilePaasNss,
		pr.ReconcileGroups,
		pr.EnsureLdapGroups,
		pr.reconcileRolebindings,
	}

	for _, reconciler := range reconcilers {
		if err = reconciler(ctx, paas); err != nil {
			return ctrl.Result{}, errors.Join(err, pr.setErrorCondition(ctx, paas, err))
		}
	}

	if err = pr.ensureAppSetCaps(ctx, paas); err != nil {
		return ctrl.Result{}, errors.Join(err, pr.setErrorCondition(ctx, paas, err))
	}

	// Reconciling succeeded, set appropriate Condition
	return ctrl.Result{}, pr.setSuccesfullCondition(ctx, paas)
}

func (pr *PaasReconciler) setSuccesfullCondition(ctx context.Context, paas *v1alpha1.Paas) error {
	meta.SetStatusCondition(&paas.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeReadyPaas,
		Status: metav1.ConditionTrue, Reason: "Reconciling", ObservedGeneration: paas.Generation,
		Message: fmt.Sprintf("Reconciled (%s) successfully", paas.Name),
	})
	meta.SetStatusCondition(&paas.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeHasErrorsPaas,
		Status: metav1.ConditionFalse, Reason: "Reconciling", ObservedGeneration: paas.Generation,
		Message: fmt.Sprintf("Reconciled (%s) successfully", paas.Name),
	})

	return pr.Status().Update(ctx, paas)
}

func (pr *PaasReconciler) setErrorCondition(ctx context.Context, paas *v1alpha1.Paas, err error) error {
	meta.SetStatusCondition(&paas.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeReadyPaas,
		Status: metav1.ConditionFalse, Reason: "ReconcilingError", ObservedGeneration: paas.Generation,
		Message: fmt.Sprintf("Reconciling (%s) failed", paas.Name),
	})
	meta.SetStatusCondition(&paas.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeHasErrorsPaas,
		Status: metav1.ConditionTrue, Reason: "ReconcilingError", ObservedGeneration: paas.Generation,
		Message: err.Error(),
	})

	return pr.Status().Update(ctx, paas)
}

// SetupWithManager sets up the controller with the Manager.
func (pr *PaasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Paas{}, builder.WithPredicates(
			predicate.Or(
				// Spec updated
				predicate.GenerationChangedPredicate{},
				// Labels updated
				predicate.LabelChangedPredicate{},
			))).
		Watches(
			&v1alpha1.PaasConfig{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				// Enqueue all Paas objects
				var reqs []reconcile.Request
				var paasList v1alpha1.PaasList
				if err := mgr.GetClient().List(context.Background(), &paasList); err != nil {
					mgr.GetLogger().Error(err, "unable to list paases")
					return nil
				}

				for _, p := range paasList.Items {
					reqs = append(reqs, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: p.Namespace,
							Name:      p.Name,
						},
					})
				}

				return reqs
			}),
			builder.WithPredicates(v1alpha1.ActivePaasConfigUpdated()),
		).
		Complete(pr)
}

func (pr *PaasReconciler) finalizePaas(ctx context.Context, paas *v1alpha1.Paas) error {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("inside Paas finalizer")
	if err := pr.FinalizeClusterQuotas(ctx, paas); err != nil {
		logger.Err(err).Msg("quota finalizer error")
		return err
	} else if err = pr.FinalizeGroups(ctx, paas); err != nil {
		logger.Err(err).Msg("group finalizer error")
		return err
	} else if err = pr.FinalizeExtraClusterRoleBindings(ctx, paas); err != nil {
		logger.Err(err).Msg("extra ClusterRoleBindings finalizer error")
		return err
	} else if err = pr.FinalizeClusterWideQuotas(ctx, paas); err != nil {
		return err
	}
	logger.Info().Msg("paaS successfully finalized")
	return nil
}

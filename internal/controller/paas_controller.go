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

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/logging"
	"github.com/belastingdienst/opr-paas/internal/paasresource"

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

// GetScheme is a simple getter for the Scheme of the Paas Controller logic
func (r PaasReconciler) GetScheme() *runtime.Scheme {
	return r.Scheme
}

// Reconciler reconciles a Paas object
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

// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns/finalizers,verbs=update

// It is advised to reduce the scope of this permission by stating the resourceNames of the roles you would like Paas to bind to, in your deployment role.yaml
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=bind
//revive:enable:line-length-limit

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Paas object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//

func (r *PaasReconciler) getPaasFromRequest(
	ctx context.Context,
	req ctrl.Request,
) (paas *v1alpha1.Paas, err error) {
	paas = &v1alpha1.Paas{}
	ctx, logger := logging.GetLogComponent(ctx, "paas")
	if err = r.Get(ctx, req.NamespacedName, paas); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	if paas.GetDeletionTimestamp() != nil {
		logger.Info().Msg("paas marked for deletion")
		return nil, r.updateFinalizer(ctx, paas)
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
	if err = r.Status().Update(ctx, paas); err != nil {
		logger.Err(err).Msg("failed to update Paas status")
		return nil, err
	}

	if err := r.Get(ctx, req.NamespacedName, paas); err != nil {
		logger.Err(err).Msg("failed to re-fetch Paas")
		return nil, err
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(paas, paasFinalizer) {
		logger.Info().Msg("paas has no finalizer yet")
		if ok := controllerutil.AddFinalizer(paas, paasFinalizer); !ok {
			logger.Err(err).Msg("failed to add finalizer")
			return nil, errors.New("failed to add finalizer")
		}
		if err := r.Update(ctx, paas); err != nil {
			logger.Err(err).Msg("error updating Paas")
			return nil, err
		}
		logger.Info().Msg("added finalizer to Paas")
	}

	return paas, nil
}

func (r *PaasReconciler) updateFinalizer(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	logger := log.Ctx(ctx)

	if controllerutil.ContainsFinalizer(paas, paasFinalizer) {
		logger.Info().Msg("finalizing Paas")
		// Let's add here a status "Downgrade" to reflect that this resource began its process to be terminated.
		meta.SetStatusCondition(paas.GetConditions(), metav1.Condition{
			Type:   v1alpha1.TypeDegradedPaas,
			Status: metav1.ConditionUnknown, Reason: "Finalizing", ObservedGeneration: paas.GetGeneration(),
			Message: fmt.Sprintf("Performing finalizer operations for Paas: %s ", paas.GetName()),
		})

		if err := r.Status().Update(ctx, paas); err != nil {
			logger.Err(err).Msg("Failed to update Paas status")
			return err
		}
		// Run finalization logic for paasFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.finalizePaas(ctx, paas); err != nil {
			return err
		}

		meta.SetStatusCondition(paas.GetConditions(), metav1.Condition{
			Type:   v1alpha1.TypeDegradedPaas,
			Status: metav1.ConditionTrue, Reason: "Finalizing", ObservedGeneration: paas.GetGeneration(),
			Message: fmt.Sprintf("Finalizer operations for Paas %s name were successfully accomplished",
				paas.GetName()),
		})

		if err := r.Status().Update(ctx, paas); err != nil {
			logger.Err(err).Msg("Failed to update Paas status")
			return err
		}

		logger.Info().Msg("removing finalizer")
		// Remove paasFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		controllerutil.RemoveFinalizer(paas, paasFinalizer)
		if err := r.Update(ctx, paas); err != nil {
			return err
		}
		logger.Info().Msg("finalization finished")
	}

	return nil
}

// Reconcile is the main entrypoint for Reconcilliation of a Paas resource
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (r *PaasReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	paas := &v1alpha1.Paas{ObjectMeta: metav1.ObjectMeta{Name: req.Name}}
	ctx, logger := logging.SetControllerLogger(ctx, paas, r.Scheme, req)

	if paas, err = r.getPaasFromRequest(ctx, req); err != nil {
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

	paasReconcilers := []func(context.Context, *v1alpha1.Paas) error{
		r.reconcileQuotas,
		r.reconcileClusterWideQuota,
		r.reconcileGroups,
		r.ensureLdapGroups,
	}

	for _, reconciler := range paasReconcilers {
		if err = reconciler(ctx, paas); err != nil {
			return ctrl.Result{}, errors.Join(err, r.setErrorCondition(ctx, paas, err))
		}
	}
	nsDefs, err := r.nsDefsFromPaas(ctx, paas)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Debug().Msgf("Need to create resourced for %d namespaces", len(nsDefs))
	paasNsReconcilers := []func(context.Context, *v1alpha1.Paas, namespaceDefs) error{
		r.reconcileNamespaces,
		r.reconcilePaasRolebindings,
		r.reconcilePaasSecrets,
		r.reconcileExtraClusterRoleBindings,
	}
	for _, reconciler := range paasNsReconcilers {
		if err = reconciler(ctx, paas, nsDefs); err != nil {
			return ctrl.Result{}, errors.Join(err, r.setErrorCondition(ctx, paas, err))
		}
	}

	if err = r.ensureAppSetCaps(ctx, paas); err != nil {
		return ctrl.Result{}, errors.Join(err, r.setErrorCondition(ctx, paas, err))
	}

	// Reconciling succeeded, set appropriate Condition
	return ctrl.Result{}, r.setSuccesfullCondition(ctx, paas)
}

func (r *PaasReconciler) setSuccesfullCondition(ctx context.Context, paas *v1alpha1.Paas) error {
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

	return r.Status().Update(ctx, paas)
}

func (r *PaasReconciler) setErrorCondition(ctx context.Context, resource paasresource.Resource, err error) error {
	meta.SetStatusCondition(resource.GetConditions(), metav1.Condition{
		Type:   v1alpha1.TypeReadyPaas,
		Status: metav1.ConditionFalse, Reason: "ReconcilingError", ObservedGeneration: resource.GetGeneration(),
		Message: fmt.Sprintf("Reconciling (%s) failed", resource.GetName()),
	})
	meta.SetStatusCondition(resource.GetConditions(), metav1.Condition{
		Type:   v1alpha1.TypeHasErrorsPaas,
		Status: metav1.ConditionTrue, Reason: "ReconcilingError", ObservedGeneration: resource.GetGeneration(),
		Message: err.Error(),
	})
	return r.Status().Update(ctx, resource)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PaasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Paas{}, builder.WithPredicates(
			predicate.Or(
				// Spec updated
				predicate.GenerationChangedPredicate{},
				// Labels updated
				predicate.LabelChangedPredicate{},
			))).
		Owns(&v1alpha1.PaasNS{}).
		Watches(
			&v1alpha1.PaasNS{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, obj client.Object) []reconcile.Request {
				// Enqueue all Paas objects
				var reqs []reconcile.Request
				var ns corev1.Namespace
				logger := mgr.GetLogger()
				if err := mgr.GetClient().Get(
					context.Background(),
					types.NamespacedName{Name: obj.GetNamespace()},
					&ns,
				); err != nil {
					logger.Error(err, "unable to get namespace where paasns resides")
					return nil
				}
				var paasNames []string
				for _, ref := range ns.OwnerReferences {
					if ref.Kind == "Paas" && *ref.Controller {
						paasNames = append(paasNames, ref.Name)
					}
				}
				if len(paasNames) == 0 {
					logger.Error(
						errors.New("failed to get owner reference with kind paas and controller=true from namespace resource"),
						"finding paas for paasns without owner reference",
						"ns",
						ns,
					)
				} else if len(paasNames) > 1 {
					logger.Error(
						errors.New("found multiple owner references with kind paas and controller=true"),
						"finding paas for paasns without owner reference",
						"ns",
						ns,
					)
				}
				paasName := paasNames[0]
				if !strings.HasPrefix(ns.Name, paasName+"-") {
					logger.Error(
						errors.New("namespace is not prefixed with paasName in owner reference"),
						"finding paas for paasns without owner reference",
						"ns",
						ns,
					)

				}

				reqs = append(reqs, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: paasName,
					},
				})

				return reqs
			}),
			builder.WithPredicates(v1alpha1.ActivePaasConfigUpdated()),
		).
		Watches(
			&v1alpha1.PaasConfig{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, _ client.Object) []reconcile.Request {
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
		Complete(r)
}

func (r *PaasReconciler) finalizePaas(ctx context.Context, paas *v1alpha1.Paas) error {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("inside Paas finalizer")

	paasReconcilers := []func(context.Context, *v1alpha1.Paas) error{
		//r.finalizeClusterQuotas,
		r.finalizeGroups,
		r.finalizeExtraClusterRoleBindings,
		r.finalizeClusterWideQuotas,
		r.finalizeAppSetCaps,
	}

	for _, reconciler := range paasReconcilers {
		if err := reconciler(ctx, paas); err != nil {
			return errors.Join(err, r.setErrorCondition(ctx, paas, err))
		}
	}

	logger.Info().Msg("paaS successfully finalized")
	return nil
}

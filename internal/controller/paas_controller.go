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

	quotav1 "github.com/openshift/api/quota/v1"
	userv1 "github.com/openshift/api/user/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/logging"
	"github.com/belastingdienst/opr-paas/v3/internal/paasresource"

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
func (r PaasReconciler) getScheme() *runtime.Scheme {
	return r.Scheme
}

// Reconciler reconciles a Paas object
type Reconciler interface {
	Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error
	Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
	getScheme() *runtime.Scheme
	Delete(context.Context, client.Object, ...client.DeleteOption) error
}

//revive:disable:line-length-limit
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas/finalizers,verbs=update

// +kubebuilder:rbac:groups=quota.openshift.io,resources=clusterresourcequotas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=user.openshift.io,resources=groups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=core,resources=secrets;namespaces,verbs=create;delete;get;list;patch;update;watch
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
) (paas *v1alpha2.Paas, err error) {
	paas = &v1alpha2.Paas{}
	_, logger := logging.GetLogComponent(ctx, logging.ControllerPaasComponent)
	if err = r.Get(ctx, req.NamespacedName, paas); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	if paas.GetDeletionTimestamp() != nil {
		logger.Info().Msg("paas marked for deletion")
		if !controllerutil.ContainsFinalizer(paas, paasFinalizer) {
			return nil, nil
		}
		for _, finalizationFunc := range []func(context.Context, *v1alpha2.Paas) error{
			r.setFinalizing,
			r.finalizePaas,
			r.removeFinalizer,
		} {
			if err = finalizationFunc(ctx, paas); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}

	// Started reconciling, reset status
	meta.SetStatusCondition(
		&paas.Status.Conditions,
		metav1.Condition{
			Type:               v1alpha2.TypeReadyPaas,
			Status:             metav1.ConditionUnknown,
			ObservedGeneration: paas.Generation,
			Reason:             "Reconciling",
			Message:            "Starting reconciliation",
		},
	)
	meta.RemoveStatusCondition(&paas.Status.Conditions, v1alpha2.TypeHasErrorsPaas)
	if err = r.Status().Update(ctx, paas); err != nil {
		logger.Err(err).Msg("failed to update Paas status")
		return nil, err
	}

	if err = r.Get(ctx, req.NamespacedName, paas); err != nil {
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
		if err = r.Update(ctx, paas); err != nil {
			logger.Err(err).Msg("error updating Paas")
			return nil, err
		}
		logger.Info().Msg("added finalizer to Paas")
	}

	return paas, nil
}

func (r *PaasReconciler) setFinalizing(
	ctx context.Context,
	paas *v1alpha2.Paas,
) error {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerPaasComponent)

	logger.Info().Msg("finalizing Paas")
	// Let's add here a status "Downgrade" to reflect that this resource began its process to be terminated.
	meta.SetStatusCondition(paas.GetConditions(), metav1.Condition{
		Type:   v1alpha2.TypeDegradedPaas,
		Status: metav1.ConditionUnknown, Reason: "Finalizing", ObservedGeneration: paas.GetGeneration(),
		Message: fmt.Sprintf("Performing finalizer operations for Paas: %s ", paas.GetName()),
	})

	if err := r.Status().Update(ctx, paas); err != nil {
		logger.Err(err).Msg("Failed to update Paas status")
		return err
	}
	return nil
}

func (r *PaasReconciler) removeFinalizer(
	ctx context.Context,
	paas *v1alpha2.Paas,
) error {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerPaasComponent)
	meta.SetStatusCondition(paas.GetConditions(), metav1.Condition{
		Type:   v1alpha2.TypeDegradedPaas,
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

	return nil
}

// Reconcile is the main entrypoint for Reconciliation of a Paas resource
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (r *PaasReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	paas := &v1alpha2.Paas{ObjectMeta: metav1.ObjectMeta{Name: req.Name}}
	ctx, logger := logging.SetControllerLogger(ctx, paas, r.Scheme, req)

	if paas, err = r.getPaasFromRequest(ctx, req); err != nil {
		logger.Err(err).Msg("could not get Paas from k8s")
		return ctrl.Result{}, err
	}

	if paas == nil {
		// r.GetPaas handled all logic and returned a nil object
		return ctrl.Result{}, nil
	}

	paasReconcilers := []func(context.Context, *v1alpha2.Paas) error{
		r.reconcileQuotas,
		r.reconcileClusterWideQuota,
		r.reconcileNamespacedResources,
		r.reconcileGroups,
		r.ensureAppSetCaps,
		r.finalizeDisabledAppSetCaps,
	}

	for _, reconciler := range paasReconcilers {
		if err = reconciler(ctx, paas); err != nil {
			return ctrl.Result{}, errors.Join(err, r.setErrorCondition(ctx, paas, err))
		}
	}
	// Reconciling succeeded, set appropriate Condition
	return ctrl.Result{}, r.setSuccessfulCondition(ctx, paas)
}

func (r *PaasReconciler) reconcileNamespacedResources(
	ctx context.Context,
	paas *v1alpha2.Paas,
) (err error) {
	_, logger := logging.GetLogComponent(ctx, logging.ControllerPaasComponent)
	logger.Debug().Msg("inside namespaced resource reconciler")
	nsDefs, err := r.nsDefsFromPaas(ctx, paas)
	if err != nil {
		return err
	}
	logger.Debug().Msgf("Need to manage resources for %d namespaces", len(nsDefs))
	paasNsReconcilers := []func(context.Context, *v1alpha2.Paas, namespaceDefs) error{
		r.reconcileNamespaces,
		r.finalizeObsoleteNamespaces,
		r.reconcilePaasRolebindings,
		r.reconcilePaasSecrets,
		r.reconcileClusterRoleBindings,
	}
	for _, reconciler := range paasNsReconcilers {
		if err = reconciler(ctx, paas, nsDefs); err != nil {
			return errors.Join(err, r.setErrorCondition(ctx, paas, err))
		}
	}

	// Reconciling succeeded, set appropriate Condition
	return r.setSuccessfulCondition(ctx, paas)
}

func (r *PaasReconciler) setSuccessfulCondition(ctx context.Context, paas *v1alpha2.Paas) error {
	meta.SetStatusCondition(&paas.Status.Conditions, metav1.Condition{
		Type:   v1alpha2.TypeReadyPaas,
		Status: metav1.ConditionTrue, Reason: "Reconciling", ObservedGeneration: paas.Generation,
		Message: fmt.Sprintf("Reconciled (%s) successfully", paas.Name),
	})
	meta.SetStatusCondition(&paas.Status.Conditions, metav1.Condition{
		Type:   v1alpha2.TypeHasErrorsPaas,
		Status: metav1.ConditionFalse, Reason: "Reconciling", ObservedGeneration: paas.Generation,
		Message: fmt.Sprintf("Reconciled (%s) successfully", paas.Name),
	})

	return r.Status().Update(ctx, paas)
}

func (r *PaasReconciler) setErrorCondition(ctx context.Context, resource paasresource.Resource, err error) error {
	meta.SetStatusCondition(resource.GetConditions(), metav1.Condition{
		Type:   v1alpha2.TypeReadyPaas,
		Status: metav1.ConditionFalse, Reason: "ReconcilingError", ObservedGeneration: resource.GetGeneration(),
		Message: fmt.Sprintf("Reconciling (%s) failed", resource.GetName()),
	})
	meta.SetStatusCondition(resource.GetConditions(), metav1.Condition{
		Type:   v1alpha2.TypeHasErrorsPaas,
		Status: metav1.ConditionTrue, Reason: "ReconcilingError", ObservedGeneration: resource.GetGeneration(),
		Message: err.Error(),
	})
	return r.Status().Update(ctx, resource)
}

func paasFromNs(ns corev1.Namespace) (string, error) {
	var paasNames []string
	for _, ref := range ns.OwnerReferences {
		if ref.Kind == "Paas" && *ref.Controller {
			paasNames = append(paasNames, ref.Name)
		}
	}
	if len(paasNames) == 0 {
		return "", errors.New("failed to get owner reference with kind paas and controller=true from namespace")
	} else if len(paasNames) > 1 {
		return "", errors.New("found multiple owner references with kind paas and controller=true")
	}
	paasName := paasNames[0]
	if !strings.HasPrefix(ns.Name, paasName+"-") {
		return "", errors.New("namespace is not prefixed with paasName in owner reference")
	}
	return paasName, nil
}

// allPaases is a simple wrapper to collect all Paas'es and created requests for them on PaasConfig changes
// allPaases is not unittests ATM. We might add an e2e test for this instead.
func allPaases(mgr ctrl.Manager) []reconcile.Request {
	// Enqueue all Paas objects
	var reqs []reconcile.Request
	var paasList v1alpha2.PaasList
	ctx := context.Background()
	_, logger := logging.GetLogComponent(ctx, logging.ControllerPaasComponent)
	if err := mgr.GetClient().List(ctx, &paasList); err != nil {
		logger.Error().AnErr("error", err).Msg("unable to list paases")
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
}

// specOrLabelsChangedPredicate returns a reusable predicate for spec or label changes
func specOrLabelsChangedPredicate() predicate.Predicate {
	return predicate.Or(
		// Spec updated
		predicate.GenerationChangedPredicate{},
		// Labels updated
		predicate.LabelChangedPredicate{},
	)
}

// SetupWithManager sets up the controller with the Manager.
// SetupWithManager is not unit-tested ATM. Mostly covered by e2e-tests.
func (r *PaasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.Paas{}, builder.WithPredicates(specOrLabelsChangedPredicate())).
		// Reconcile on owned resources changes
		Owns(&quotav1.ClusterResourceQuota{}, builder.WithPredicates(specOrLabelsChangedPredicate())).
		Owns(&userv1.Group{}, builder.WithPredicates(specOrLabelsChangedPredicate())).
		Owns(&corev1.Secret{}, builder.WithPredicates(specOrLabelsChangedPredicate())).
		Owns(&corev1.Namespace{}, builder.WithPredicates(specOrLabelsChangedPredicate())).
		Owns(&rbacv1.RoleBinding{}, builder.WithPredicates(specOrLabelsChangedPredicate())).
		Owns(&rbacv1.ClusterRoleBinding{}, builder.WithPredicates(specOrLabelsChangedPredicate())).
		// TODO(portly-halicore-76):We don't own PaasNS objects correctly yet
		// Owns(&v1alpha2.PaasNS{}, builder.WithPredicates(specOrLabelsChangedPredicate())).
		// TODO(portly-halicore-76): We don't own Rolebinding objects correctly yet
		// Owns(&rbac.RoleBinding{}, builder.WithPredicates(specOrLabelsChangedPredicate())).
		Watches(
			&v1alpha2.PaasNS{},
			handler.EnqueueRequestsFromMapFunc(
				func(ctx context.Context, paasNsObj client.Object) []reconcile.Request {
					paasName, err := r.getPaasNameFromPaasNs(ctx, paasNsObj)
					if err != nil {
						return nil
					}
					return []reconcile.Request{{
						NamespacedName: types.NamespacedName{Name: paasName},
					}}
				},
			), builder.WithPredicates(specOrLabelsChangedPredicate())).
		Watches(
			&v1alpha2.PaasConfig{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, _ client.Object) []reconcile.Request {
				return allPaases(mgr)
			}),
			builder.WithPredicates(v1alpha2.ActivePaasConfigUpdated()),
		).
		Complete(r)
}

func (r *PaasReconciler) finalizePaas(ctx context.Context, paas *v1alpha2.Paas) error {
	_, logger := logging.GetLogComponent(ctx, logging.ControllerPaasComponent)
	logger.Debug().Msg("inside Paas finalizer")

	paasReconcilers := []func(context.Context, *v1alpha2.Paas) error{
		r.finalizeGroups,
		r.finalizePaasClusterRoleBindings,
		r.finalizeClusterWideQuotas,
		r.finalizeAllAppSetCaps,
	}

	for _, reconciler := range paasReconcilers {
		if err := reconciler(ctx, paas); err != nil {
			return errors.Join(err, r.setErrorCondition(ctx, paas, err))
		}
	}

	logger.Info().Msg("paaS successfully finalized")
	return nil
}

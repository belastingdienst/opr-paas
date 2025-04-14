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

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	paasNsFinalizer     = "paasns.cpet.belastingdienst.nl/finalizer"
	paasNsComponentName = "paasns"
)

// PaasNSReconciler reconciles a PaasNS object
type PaasNSReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the PaasNS object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//

func (r *PaasReconciler) getPaasNS(ctx context.Context, req ctrl.Request) (paasns *v1alpha1.PaasNS, err error) {
	paasns = &v1alpha1.PaasNS{}
	ctx, logger := logging.GetLogComponent(ctx, paasNsComponentName)
	logger.Info().Msg("reconciling PaasNs")

	if err = r.Get(ctx, req.NamespacedName, paasns); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	// Started reconciling, reset status
	meta.SetStatusCondition(
		&paasns.Status.Conditions,
		metav1.Condition{
			Type:               v1alpha1.TypeReadyPaasNs,
			Status:             metav1.ConditionUnknown,
			ObservedGeneration: paasns.Generation,
			Reason:             "Reconciling",
			Message:            "Starting reconciliation",
		},
	)
	meta.RemoveStatusCondition(&paasns.Status.Conditions, v1alpha1.TypeHasErrorsPaasNs)
	if err = r.Status().Update(ctx, paasns); err != nil {
		logger.Err(err).Msg("Failed to update PaasNs status")
		return nil, err
	}

	if err := r.Get(ctx, req.NamespacedName, paasns); err != nil {
		logger.Err(err).Msg("failed to re-fetch PaasNs")
		return nil, err
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(paasns, paasNsFinalizer) {
		logger.Info().Msg("paasNs object has no finalizer yet")
		if ok := controllerutil.AddFinalizer(paasns, paasNsFinalizer); !ok {
			logger.Error().Msg("failed to add finalizer")
			return nil, errors.New("failed to add finalizer")
		}
		if err := r.Update(ctx, paasns); err != nil {
			logger.Err(err).Msg("error updating PaasNs")
			return nil, err
		}
		logger.Info().Msg("added finalizer to PaasNs")
	}

	// TODO(portly-halicore-76) Move to admission webhook once available
	// check if Config is set, as reconciling and finalizing without config, leaves object in limbo.
	// This is only an issue when object is being removed.
	// Finalizers will not be removed causing the object to be in limbo.
	if reflect.DeepEqual(v1alpha1.PaasConfigSpec{}, config.GetConfig().Spec) {
		logger.Error().Msg(noConfigFoundMsg)
		err = r.setErrorCondition(
			ctx,
			paasns,
			errors.New(
				// revive:disable-next-line
				"please reach out to your system administrator as there is no Paasconfig available to reconcile against",
			),
		)
		if err != nil {
			logger.Err(err).Msg("failed to set Error Condition")
			return nil, err
		}
		return nil, errors.New(noConfigFoundMsg)
	}

	if paasns.GetDeletionTimestamp() != nil {
		logger.Info().Msg("paasNS object marked for deletion")
		return nil, r.updateFinalizer(ctx, paasns)
	}

	return paasns, nil
}

func (r *PaasReconciler) getPaas(ctx context.Context, resource v1alpha1.Resource) (paas *v1alpha1.Paas, err error) {
	if paas, ok := resource.(*v1alpha1.Paas); ok {
		return paas, nil
	}
	paasns, ok := resource.(*v1alpha1.PaasNS)
	if !ok {
		return nil, fmt.Errorf("invalid resource type %T is not Paas or PaasNS", resource)
	}
	paas, _, err = r.paasFromPaasNs(ctx, paasns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = fmt.Errorf("cannot find Paas %s", paasns.Spec.Paas)
		}
		// This cannot be resolved by itself, so we should not have this keep on reconciling
		return nil, err
	}
	if !paas.AmIOwner(paasns.OwnerReferences) {
		if err := controllerutil.SetControllerReference(paas, paasns, r.Scheme); err != nil {
			return nil, err
		}
	}
	return paas, err
}

// Reconcile is the main entrypoint for Reconcilliation of a PaasNS resource
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (r *PaasReconciler) reconcilePaasNs(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	paasns := &v1alpha1.PaasNS{ObjectMeta: metav1.ObjectMeta{Name: req.Name}}
	ctx, logger := logging.SetControllerLogger(ctx, paasns, r.Scheme, req)

	if paasns, err = r.getPaasNS(ctx, req); err != nil {
		// TODO(portly-halicore-76) move to admission webhook once available
		// Don't requeue that often when no config is found
		if strings.Contains(err.Error(), noConfigFoundMsg) {
			return ctrl.Result{RequeueAfter: requeueTimeout}, nil
		}
		logger.Err(err).Msg("could not get PaasNs from k8s")
		return ctrl.Result{}, err
	}

	if paasns == nil {
		// r.getPaasNS handled all logic and returned a nil object
		return ctrl.Result{}, nil
	}

	// TODO(portly-halicore-76) remove once api version is upgraded
	paasns.Status.Truncate()

	var paas *v1alpha1.Paas
	if paas, err = r.getPaas(ctx, paasns); err != nil {
		// This cannot be resolved by itself, so we should not have this keep on reconciling,
		// only try again when setErrorCondition fails
		return ctrl.Result{}, r.setErrorCondition(ctx, paasns, err)
	}

	err = r.reconcileNamespaces(ctx, paas, paasns)
	if err != nil {
		return ctrl.Result{}, errors.Join(err, r.setErrorCondition(ctx, paasns, err))
	}

	err = r.reconcileRolebindings(ctx, paas)
	if err != nil {
		return ctrl.Result{}, errors.Join(err, r.setErrorCondition(ctx, paasns, err))
	}

	err = r.reconcileSecrets(ctx, paas, paasns)
	if err != nil {
		return ctrl.Result{}, errors.Join(err, r.setErrorCondition(ctx, paasns, err))
	}

	err = r.reconcileExtraClusterRoleBinding(ctx, paasns, paas)
	if err != nil {
		logger.Err(err).Msg("reconciling Extra ClusterRoleBindings failed")
		return ctrl.Result{}, errors.Join(err, r.setErrorCondition(ctx, paasns, err))
	}

	// Reconciling succeeded, set appropriate Condition
	return ctrl.Result{}, r.setSuccessfulCondition(ctx, paasns)
}

func (r *PaasReconciler) setSuccessfulCondition(ctx context.Context, paasNs *v1alpha1.PaasNS) error {
	meta.SetStatusCondition(&paasNs.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeReadyPaasNs,
		Status: metav1.ConditionTrue, Reason: "Reconciling", ObservedGeneration: paasNs.Generation,
		Message: fmt.Sprintf("Reconciled (%s) successfully", paasNs.Name),
	})
	meta.SetStatusCondition(&paasNs.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeHasErrorsPaasNs,
		Status: metav1.ConditionFalse, Reason: "Reconciling", ObservedGeneration: paasNs.Generation,
		Message: fmt.Sprintf("Reconciled (%s) successfully", paasNs.Name),
	})

	return r.Status().Update(ctx, paasNs)
}

// nssFromNs gets all PaasNs objects from a namespace and returns a list of all the corresponding namespaces
// It also returns PaasNS in those namespaces recursively.
func (r *PaasReconciler) nssFromNs(ctx context.Context, ns string) map[string]int {
	nss := make(map[string]int)
	pnsList := &v1alpha1.PaasNSList{}
	if err := r.List(ctx, pnsList, &client.ListOptions{Namespace: ns}); err != nil {
		// In this case panic is ok, since this situation can only occur when either k8s is down,
		// or permissions are insufficient. Both cases we should not continue executing code...
		panic(err)
	}
	for _, pns := range pnsList.Items {
		nsName := pns.NamespaceName()
		if value, exists := nss[nsName]; exists {
			nss[nsName] = value + 1
		} else {
			nss[nsName] = 1
		}
		// Call myself (recursively)
		for key, value := range r.nssFromNs(ctx, nsName) {
			nss[key] += value
		}
	}
	return nss
}

// nsFromPaas accepts a Paas and returns a list of all namespaces managed by this Paas
// nsFromPaas uses nsFromNs which is recursive.
func (r *PaasReconciler) nssFromPaas(ctx context.Context, paas *v1alpha1.Paas) map[string]int {
	finalNss := make(map[string]int)
	finalNss[paas.Name] = 1
	for key, value := range r.nssFromNs(ctx, paas.Name) {
		finalNss[key] += value
	}
	return finalNss
}

// nsFromNs gets all PaasNs objects from a namespace and returns a list of all the corresponding namespaces
// It also returns PaasNS in those namespaces recursively.
func (r *PaasReconciler) pnsFromNs(ctx context.Context, ns string) map[string]v1alpha1.PaasNS {
	nss := make(map[string]v1alpha1.PaasNS)
	pnsList := &v1alpha1.PaasNSList{}
	if err := r.List(ctx, pnsList, &client.ListOptions{Namespace: ns}); err != nil {
		return nss
	}
	for _, pns := range pnsList.Items {
		nsName := pns.NamespaceName()
		if _, exists := nss[nsName]; !exists {
			nss[nsName] = pns
			// Call myself (recursiveness)
			for key, value := range r.pnsFromNs(ctx, nsName) {
				nss[key] = value
			}
		} else {
			nss[nsName] = pns
		}
	}
	return nss
}

func (r *PaasReconciler) paasFromPaasNs(
	ctx context.Context,
	paasns *v1alpha1.PaasNS,
) (paas *v1alpha1.Paas, namespaces map[string]int, err error) {
	ctx, logger := logging.GetLogComponent(ctx, paasNsComponentName)
	paas = &v1alpha1.Paas{}
	if err := r.Get(ctx, types.NamespacedName{Name: paasns.Spec.Paas}, paas); err != nil {
		logger.Err(err).Msg("cannot get Paas")
		return nil, namespaces, err
	}
	if paasns.Namespace == paas.Name {
		return paas, r.nssFromPaas(ctx, paas), nil
	}
	namespaces = r.nssFromPaas(ctx, paas)
	if _, exists := namespaces[paasns.Namespace]; exists {
		return paas, namespaces, nil
	}
	var nss []string
	for key := range namespaces {
		nss = append(nss, key)
	}
	err = fmt.Errorf(
		"PaasNs %s claims to come from paas %s, but %s is not in the list of namespaces coming from %s (%s)",
		types.NamespacedName{Name: paasns.Name, Namespace: paasns.Namespace},
		paas.Name,
		paasns.Namespace,
		paas.Name,
		strings.Join(nss, ", "))
	return nil, map[string]int{}, err
}

func (r *PaasReconciler) finalizePaasNs(ctx context.Context, paasns *v1alpha1.PaasNS) error {
	ctx, logger := logging.GetLogComponent(ctx, paasNsComponentName)

	cfg := config.GetConfig().Spec
	// If PaasNs is related to a capability, remove it from appSet
	if _, exists := cfg.Capabilities[paasns.Name]; exists {
		if err := r.finalizeAppSetCap(ctx, paasns); err != nil {
			err = fmt.Errorf(
				"cannot remove paas from capability ApplicationSet belonging to Paas %s: %s",
				paasns.Spec.Paas,
				err.Error(),
			)
			return err
		}
	}

	paas, nss, err := r.paasFromPaasNs(ctx, paasns)
	if err != nil {
		err = fmt.Errorf("cannot find Paas %s: %s", paasns.Spec.Paas, err.Error())
		logger.Info().Msg(err.Error())
		return nil
	} else if nss[paasns.NamespaceName()] > 1 {
		err = errors.New("this is not the only paasns managing this namespace, silently removing this paasns")
		logger.Info().Msg(err.Error())
		return nil
	}

	logger.Info().Msg("inside PaasNs finalizer")
	if err := r.finalizeNamespace(ctx, paasns, paas); err != nil {
		err = fmt.Errorf("cannot remove namespace belonging to Paas %s: %s", paasns.Spec.Paas, err.Error())
		return err
	}
	if _, isCapability := paas.Spec.Capabilities[paasns.Name]; isCapability {
		logger.Info().Msg("paasNs is a capability, also finalizing Cluster Resource Quota")
		if err := r.finalizeClusterQuota(ctx, paas, paasns.NamespaceName()); err != nil {
			logger.Err(err).Msg(fmt.Sprintf("failure while finalizing quota %s", paasns.Name))
			return err
		}
	}
	logger.Info().Msg("paasNs successfully finalized")
	return nil
}

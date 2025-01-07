/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const paasconfigFinalizer = "paasconfig.cpet.belastingdienst.nl/finalizer"

// PaasConfigReconciler reconciles a PaasConfig object
type PaasConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (prc PaasConfigReconciler) GetScheme() *runtime.Scheme {
	return prc.Scheme
}

//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasconfig,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasconfig/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasconfig/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the PaasNS object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//

// SetupWithManager sets up the controller with the Manager.
func (r *PaasConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PaasConfig{}).
		WithEventFilter(
			predicate.GenerationChangedPredicate{}, // Spec changed .
		).
		Complete(r)
}

func (pcr *PaasConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	config := &v1alpha1.PaasConfig{}
	ctx, logger := setRequestLogger(ctx, config, pcr.Scheme, req)
	ctx = setLogComponent(ctx, "paasconfig")

	errResult := reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Second * 10,
	}

	if err := pcr.Get(ctx, req.NamespacedName, config); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info().Msg("reconciling PaasConfig")

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(config, paasconfigFinalizer) {
		if ok := controllerutil.AddFinalizer(config, paasconfigFinalizer); !ok {
			return errResult, fmt.Errorf("failed to add finalizer")
		}
		if err := pcr.Update(ctx, config); err != nil {
			logger.Err(err).Msg("error updating PaasConfig")
			return errResult, nil
		}
		logger.Info().Msg("added finalizer to PaasConfig")
	}

	if config.GetDeletionTimestamp() != nil {
		logger.Info().Msg("paasconfig marked for deletion")
		if controllerutil.ContainsFinalizer(config, paasconfigFinalizer) {
			// Let's add here a status "Downgrade" to reflect that this resource began its process to be terminated.
			meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
				Type:   v1alpha1.TypeDegradedPaasConfig,
				Status: metav1.ConditionUnknown, Reason: "Finalizing", ObservedGeneration: config.Generation,
				Message: fmt.Sprintf("Performing finalizer operations for PaasConfig: %s ", config.Name),
			})

			if err := pcr.Status().Update(ctx, config); err != nil {
				logger.Err(err).Msg("Failed to update PaasConfig status")
				return errResult, nil
			}
			// Reset Config if this was the active config
			if meta.IsStatusConditionPresentAndEqual(config.Status.Conditions, v1alpha1.TypeActivePaasConfig, metav1.ConditionTrue) {
				SetConfig(v1alpha1.PaasConfig{})
			}

			logger.Info().Msg("config reset successfully")
			meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
				Type:   v1alpha1.TypeDegradedPaasConfig,
				Status: metav1.ConditionTrue, Reason: "Finalizing", ObservedGeneration: config.Generation,
				Message: fmt.Sprintf("Finalizer operations for PaasConfig %s name were successfully accomplished", config.Name),
			})

			if err := pcr.Status().Update(ctx, config); err != nil {
				logger.Err(err).Msg("Failed to update PaasConfig status")
				return errResult, nil
			}

			if ok := controllerutil.RemoveFinalizer(config, paasconfigFinalizer); !ok {
				return errResult, fmt.Errorf("failed to add finalizer")
			}
			if err := pcr.Update(ctx, config); err != nil {
				logger.Err(err).Msg("error updating PaasConfig")
				return errResult, nil
			}
		}
		return ctrl.Result{}, nil
	}

	configList := &v1alpha1.PaasConfigList{}
	if err := pcr.List(ctx, configList); err != nil {
		logger.Err(err).Msg("error listing PaasConfigs")
		err := pcr.setErrorCondition(ctx, config, err)
		if err != nil {
			logger.Err(err).Msg("failed to update PaasConfig status")
			return errResult, nil
		}
		return errResult, nil
	}

	// Enforce singleton pattern
	// TODO(portly-halicore-76) move to admission webhook when available
	for _, existingConfig := range configList.Items {
		if meta.IsStatusConditionPresentAndEqual(existingConfig.Status.Conditions, v1alpha1.TypeActivePaasConfig, metav1.ConditionTrue) && existingConfig.ObjectMeta.Name != config.Name {
			// There is already another config which is the active one so we don't allow adding a new one
			singletonErr := fmt.Errorf("paasConfig singleton violation")
			logger.Err(singletonErr).Msg("more than one PaasConfig instance found")
			err := pcr.setErrorCondition(ctx, config, singletonErr)
			if err != nil {
				logger.Err(err).Msg("failed to update PaasConfig status")
				return errResult, nil
			}
			// don't reconcile this one again as that won't change anything.. I guess.
			return ctrl.Result{}, nil
		}
	}

	// As there can be reasons why we reconcile again, we check if there is a diff in the desired state vs GetConfig()
	if reflect.DeepEqual(config.Spec, GetConfig()) {
		logger.Debug().Msg("Config already equals desired state")
		// Reconciling succeeded, set appropriate Condition
		err := pcr.setSuccesfullCondition(ctx, config)
		if err != nil {
			logger.Err(err).Msg("failed to update PaasConfig status")
			return errResult, nil
		}
		return ctrl.Result{}, nil
	}

	logger.Info().Msg("configuration has changed")
	// Update the shared configuration store
	SetConfig(*config)
	logger.Debug().Msg("set active PaasConfig successfully")

	// Reconciling succeeded, set appropriate Condition
	err := pcr.setSuccesfullCondition(ctx, config)
	if err != nil {
		logger.Err(err).Msg("failed to update PaasConfig status")
		return errResult, nil
	}
	return ctrl.Result{}, nil
}

func (pcr *PaasConfigReconciler) setSuccesfullCondition(ctx context.Context, config *v1alpha1.PaasConfig) error {
	meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeActivePaasConfig,
		Status: metav1.ConditionTrue, Reason: "Reconciling", ObservedGeneration: config.Generation,
		Message: "This config is the active config!",
	})
	meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeHasErrorsPaasConfig,
		Status: metav1.ConditionFalse, Reason: "Reconciling", ObservedGeneration: config.Generation,
		Message: fmt.Sprintf("Reconciled (%s) successfully", config.Name),
	})

	if err := pcr.Status().Update(ctx, config); err != nil {
		return err
	}
	return nil
}

func (pcr *PaasConfigReconciler) setErrorCondition(ctx context.Context, config *v1alpha1.PaasConfig, err error) error {
	meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeActivePaasConfig,
		Status: metav1.ConditionFalse, Reason: "ReconcilingError", ObservedGeneration: config.Generation,
		Message: fmt.Sprintf("Reconciling (%s) failed", config.Name),
	})
	meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeHasErrorsPaasConfig,
		Status: metav1.ConditionTrue, Reason: "ReconcilingError", ObservedGeneration: config.Generation,
		Message: err.Error(),
	})

	if err := pcr.Status().Update(ctx, config); err != nil {
		return err
	}
	return nil
}

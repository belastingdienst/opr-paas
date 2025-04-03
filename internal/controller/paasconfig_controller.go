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

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const paasconfigFinalizer = "paasconfig.cpet.belastingdienst.nl/finalizer"

// PaasConfigReconciler reconciles a PaasConfig object
type PaasConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (pcr PaasConfigReconciler) GetScheme() *runtime.Scheme {
	return pcr.Scheme
}

// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasconfig,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasconfig/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasconfig/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the PaasNS object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//

// SetupWithManager sets up the controller with the Manager.
func (pcr *PaasConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PaasConfig{}).
		WithEventFilter(
			predicate.GenerationChangedPredicate{}, // Spec changed .
		).
		Complete(pcr)
}

func (pcr *PaasConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cfg := &v1alpha1.PaasConfig{}
	ctx, _ = logging.SetControllerLogger(ctx, cfg, pcr.Scheme, req)
	ctx, logger := logging.GetLogComponent(ctx, "paasconfig")

	errResult := reconcile.Result{
		Requeue:      true,
		RequeueAfter: requeueTimeout,
	}

	if err := pcr.Get(ctx, req.NamespacedName, cfg); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info().Msg("reconciling PaasConfig")

	if requeue, err := pcr.addFinalizer(ctx, cfg); requeue {
		return errResult, err
	}

	if cfg.GetDeletionTimestamp() != nil {
		if requeue, err := pcr.updateFinalizer(ctx, cfg); requeue {
			return errResult, err
		}

		return ctrl.Result{}, nil
	}

	// As there can be reasons why we reconcile again, we check if there is a diff in the desired state vs GetConfig()
	if reflect.DeepEqual(cfg.Spec, config.GetConfig().Spec) {
		logger.Debug().Msg("Config already equals desired state")
		// Reconciling succeeded, set appropriate Condition
		err := pcr.setSuccessfulCondition(ctx, cfg)
		if err != nil {
			logger.Err(err).Msg("failed to update PaasConfig status")
			return errResult, nil
		}
		return ctrl.Result{}, nil
	}

	logger.Info().Msg("configuration has changed")
	if !reflect.DeepEqual(cfg.Spec.DecryptKeysSecret, config.GetConfig().Spec.DecryptKeysSecret) {
		resetCrypts()
	}
	// Update the shared configuration store
	logger.Debug().Msg("set active PaasConfig successfully")

	// Reconciling succeeded, set appropriate Condition
	err := pcr.setSuccessfulCondition(ctx, cfg)
	if err != nil {
		logger.Err(err).Msg("failed to update PaasConfig status")
		return errResult, nil
	}
	return ctrl.Result{}, nil
}

func (pcr *PaasConfigReconciler) addFinalizer(ctx context.Context, cfg *v1alpha1.PaasConfig) (requeue bool, err error) {
	logger := log.Ctx(ctx)

	if !controllerutil.ContainsFinalizer(cfg, paasconfigFinalizer) {
		if ok := controllerutil.AddFinalizer(cfg, paasconfigFinalizer); !ok {
			return true, fmt.Errorf("failed to add finalizer")
		}
		if err := pcr.Update(ctx, cfg); err != nil {
			logger.Err(err).Msg("error updating PaasConfig")
			return true, nil
		}
		logger.Info().Msg("added finalizer to PaasConfig")
	}

	return false, nil
}

func (pcr *PaasConfigReconciler) updateFinalizer(
	ctx context.Context,
	cfg *v1alpha1.PaasConfig,
) (requeue bool, err error) {
	logger := log.Ctx(ctx)
	logger.Info().Msg("paasconfig marked for deletion")

	if controllerutil.ContainsFinalizer(cfg, paasconfigFinalizer) {
		// Let's add here a status "Downgrade" to reflect that this resource began its process to be terminated.
		meta.SetStatusCondition(&cfg.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.TypeDegradedPaasConfig,
			Status: metav1.ConditionUnknown, Reason: "Finalizing", ObservedGeneration: cfg.Generation,
			Message: fmt.Sprintf("Performing finalizer operations for PaasConfig: %s ", cfg.Name),
		})

		if err := pcr.Status().Update(ctx, cfg); err != nil {
			logger.Err(err).Msg("Failed to update PaasConfig status")
			return true, nil
		}

		logger.Info().Msg("config reset successfully")
		meta.SetStatusCondition(&cfg.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.TypeDegradedPaasConfig,
			Status: metav1.ConditionTrue, Reason: "Finalizing", ObservedGeneration: cfg.Generation,
			Message: fmt.Sprintf(
				"Finalizer operations for PaasConfig %s name were successfully accomplished",
				cfg.Name,
			),
		})

		if err := pcr.Status().Update(ctx, cfg); err != nil {
			logger.Err(err).Msg("Failed to update PaasConfig status")
			return true, nil
		}

		if ok := controllerutil.RemoveFinalizer(cfg, paasconfigFinalizer); !ok {
			return true, fmt.Errorf("failed to add finalizer")
		}
		if err := pcr.Update(ctx, cfg); err != nil {
			logger.Err(err).Msg("error updating PaasConfig")
			return true, nil
		}
	}

	return false, nil
}

func (pcr *PaasConfigReconciler) setSuccessfulCondition(ctx context.Context, paasConfig *v1alpha1.PaasConfig) error {
	meta.SetStatusCondition(&paasConfig.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeActivePaasConfig,
		Status: metav1.ConditionTrue, Reason: "Reconciling", ObservedGeneration: paasConfig.Generation,
		Message: "This config is the active config!",
	})
	meta.SetStatusCondition(&paasConfig.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TypeHasErrorsPaasConfig,
		Status: metav1.ConditionFalse, Reason: "Reconciling", ObservedGeneration: paasConfig.Generation,
		Message: fmt.Sprintf("Reconciled (%s) successfully", paasConfig.Name),
	})

	return pcr.Status().Update(ctx, paasConfig)
}

/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/config"
	"github.com/belastingdienst/opr-paas/v3/internal/logging"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const paasconfigFinalizer = "paasconfig.cpet.belastingdienst.nl/finalizer"

// PaasConfigReconciler reconciles a PaasConfig object
type PaasConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// GetScheme is a simple getter for the Scheme of the PaasConfig Controller logic
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
		For(&v1alpha2.PaasConfig{}).
		WithEventFilter(
			predicate.GenerationChangedPredicate{}, // Spec changed .
		).
		Complete(pcr)
}

// Reconcile is the main entrypoint for Reconciliation of a PaasConfig resource
func (pcr *PaasConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cfg := &v1alpha2.PaasConfig{}
	ctx, _ = logging.SetControllerLogger(ctx, cfg, pcr.Scheme, req)
	ctx, logger := logging.GetLogComponent(ctx, "paasconfig")

	if err := pcr.Get(ctx, req.NamespacedName, cfg); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	meta.SetStatusCondition(
		&cfg.Status.Conditions,
		metav1.Condition{
			Type:               v1alpha2.TypeHasErrorsPaasConfig,
			Status:             metav1.ConditionUnknown,
			ObservedGeneration: cfg.Generation,
			Reason:             "Reconciling",
			Message:            "Starting reconciliation",
		},
	)

	logger.Info().Msg("reconciling PaasConfig")

	if err := pcr.Status().Update(ctx, cfg); err != nil {
		logger.Err(err).Msg("failed to update PaasConfig status")
		return ctrl.Result{}, err
	}

	if requeue, err := pcr.addFinalizer(ctx, cfg); requeue {
		return ctrl.Result{}, err
	}

	if cfg.GetDeletionTimestamp() != nil {
		if requeue, err := pcr.finalize(ctx, cfg); requeue {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// As there can be reasons why we reconcile again, we check if there is a diff in the desired state vs GetConfig()
	// when there is no change, we exit this function.
	if reflect.DeepEqual(cfg.Spec, config.GetConfig().Spec) {
		logger.Info().Msg("Cached config equals desired state")
		// Reconciling succeeded, set appropriate Condition
		err := pcr.setSuccessfulCondition(ctx, cfg)
		if err != nil {
			logger.Err(err).Msg("failed to update PaasConfig status")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}

	logger.Info().Msg("configuration has changed")
	// If the decryptSecrets have been configured differently, we must reset
	// the cached crypts as those are no longer valid.
	if !reflect.DeepEqual(cfg.Spec.DecryptKeysSecret, config.GetConfig().Spec.DecryptKeysSecret) {
		logger.Info().Msg("Decryption keys changed")
		resetCrypts()
	}
	// Update the shared configuration store
	config.SetConfig(*cfg)
	logger.Info().Msg("Set the cached config successfully")

	// Reconciling succeeded, set appropriate Condition
	err := pcr.setSuccessfulCondition(ctx, cfg)
	if err != nil {
		logger.Err(err).Msg("failed to update PaasConfig status")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func (pcr *PaasConfigReconciler) addFinalizer(ctx context.Context, cfg *v1alpha2.PaasConfig) (requeue bool, err error) {
	logger := log.Ctx(ctx)

	if !controllerutil.ContainsFinalizer(cfg, paasconfigFinalizer) {
		if ok := controllerutil.AddFinalizer(cfg, paasconfigFinalizer); !ok {
			return true, errors.New("failed to add finalizer")
		}
		if err = pcr.Update(ctx, cfg); err != nil {
			logger.Err(err).Msg("error updating PaasConfig")
			return true, nil
		}
		logger.Info().Msg("added finalizer to PaasConfig")
	}

	return false, nil
}

func (pcr *PaasConfigReconciler) finalize(
	ctx context.Context,
	cfg *v1alpha2.PaasConfig,
) (requeue bool, err error) {
	logger := log.Ctx(ctx)
	logger.Info().Msg("paasconfig marked for deletion")

	if controllerutil.ContainsFinalizer(cfg, paasconfigFinalizer) {
		// Let's add here a status "Downgrade" to reflect that this resource began its process to be terminated.
		meta.SetStatusCondition(&cfg.Status.Conditions, metav1.Condition{
			Type:   v1alpha2.TypeDegradedPaasConfig,
			Status: metav1.ConditionUnknown, Reason: "Finalizing", ObservedGeneration: cfg.Generation,
			Message: fmt.Sprintf("Performing finalizer operations for PaasConfig: %s ", cfg.Name),
		})

		if err = pcr.Status().Update(ctx, cfg); err != nil {
			logger.Err(err).Msg("Failed to update PaasConfig status")
			return true, nil
		}

		logger.Info().Msg("config reset successfully")
		meta.SetStatusCondition(&cfg.Status.Conditions, metav1.Condition{
			Type:   v1alpha2.TypeDegradedPaasConfig,
			Status: metav1.ConditionTrue, Reason: "Finalizing", ObservedGeneration: cfg.Generation,
			Message: fmt.Sprintf(
				"Finalizer operations for PaasConfig %s name were successfully accomplished",
				cfg.Name,
			),
		})

		if err = pcr.Status().Update(ctx, cfg); err != nil {
			logger.Err(err).Msg("Failed to update PaasConfig status")
			return true, nil
		}

		if ok := controllerutil.RemoveFinalizer(cfg, paasconfigFinalizer); !ok {
			return true, errors.New("failed to add finalizer")
		}
		if err = pcr.Update(ctx, cfg); err != nil {
			logger.Err(err).Msg("error updating PaasConfig")
			return true, nil
		}
	}

	return false, nil
}

func (pcr *PaasConfigReconciler) setSuccessfulCondition(ctx context.Context, paasConfig *v1alpha2.PaasConfig) error {
	meta.SetStatusCondition(&paasConfig.Status.Conditions, metav1.Condition{
		Type:   v1alpha2.TypeActivePaasConfig,
		Status: metav1.ConditionTrue, Reason: "Reconciling", ObservedGeneration: paasConfig.Generation,
		Message: "This config is the active config!",
	})
	meta.SetStatusCondition(&paasConfig.Status.Conditions, metav1.Condition{
		Type:   v1alpha2.TypeHasErrorsPaasConfig,
		Status: metav1.ConditionFalse, Reason: "Reconciling", ObservedGeneration: paasConfig.Generation,
		Message: fmt.Sprintf("Reconciled (%s) successfully", paasConfig.Name),
	})

	return pcr.Status().Update(ctx, paasConfig)
}

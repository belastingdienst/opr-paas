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
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"
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
	cfg := &v1alpha1.PaasConfig{}
	ctx, _ = logging.SetControllerLogger(ctx, cfg, pcr.Scheme, req)
	ctx, logger := logging.GetLogComponent(ctx, "paasconfig")

	errResult := reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Second * 10,
	}

	if err := pcr.Get(ctx, req.NamespacedName, cfg); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info().Msg("reconciling PaasConfig")

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(cfg, paasconfigFinalizer) {
		if ok := controllerutil.AddFinalizer(cfg, paasconfigFinalizer); !ok {
			return errResult, fmt.Errorf("failed to add finalizer")
		}
		if err := pcr.Update(ctx, cfg); err != nil {
			logger.Err(err).Msg("error updating PaasConfig")
			return errResult, nil
		}
		logger.Info().Msg("added finalizer to PaasConfig")
	}

	if cfg.GetDeletionTimestamp() != nil {
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
				return errResult, nil
			}
			// Reset Config if this was the active config
			if meta.IsStatusConditionPresentAndEqual(
				cfg.Status.Conditions,
				v1alpha1.TypeActivePaasConfig,
				metav1.ConditionTrue,
			) {
				config.SetConfig(v1alpha1.PaasConfig{})
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
				return errResult, nil
			}

			if ok := controllerutil.RemoveFinalizer(cfg, paasconfigFinalizer); !ok {
				return errResult, fmt.Errorf("failed to add finalizer")
			}
			if err := pcr.Update(ctx, cfg); err != nil {
				logger.Err(err).Msg("error updating PaasConfig")
				return errResult, nil
			}
		}
		return ctrl.Result{}, nil
	}

	// As there can be reasons why we reconcile again, we check if there is a diff in the desired state vs GetConfig()
	if reflect.DeepEqual(cfg.Spec, config.GetConfig()) {
		logger.Debug().Msg("Config already equals desired state")
		// Reconciling succeeded, set appropriate Condition
		err := pcr.setSuccesfullCondition(ctx, cfg)
		if err != nil {
			logger.Err(err).Msg("failed to update PaasConfig status")
			return errResult, nil
		}
		return ctrl.Result{}, nil
	}

	logger.Info().Msg("configuration has changed")
	if !reflect.DeepEqual(cfg.Spec.DecryptKeysSecret, config.GetConfig().DecryptKeysSecret) {
		resetCrypts()
	}
	// Update the shared configuration store
	config.SetConfig(*cfg)
	logger.Debug().Msg("set active PaasConfig successfully")

	// Reconciling succeeded, set appropriate Condition
	err := pcr.setSuccesfullCondition(ctx, cfg)
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

/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const paasconfigFinalizer = "paasconfig.cpet.belastingdienst.nl/finalizer"

// PaasConfigReconciler reconciles a PaasConfig object
type PaasConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
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
			predicate.Or(
				predicate.GenerationChangedPredicate{}, // Spec changed
				// TODO add custom predicate funcs for more finegrained reconciliation?
			)).
		Complete(r)
}

func (pcr *PaasConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = setLogComponent(ctx, "paasconfig")
	logger := log.Ctx(ctx)
	logger.Info().Msg("reconciling PaasConfig")

	// Fetch the singleton PaasConfig instance
	config := &v1alpha1.PaasConfig{}
	if err := pcr.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, config); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(config, paasconfigFinalizer) {
		if ok := controllerutil.AddFinalizer(config, paasconfigFinalizer); !ok {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer")
		}
		if err := pcr.Update(ctx, config); err != nil {
			logger.Err(err).Msg("error updating PaasConfig")
			return ctrl.Result{}, err
		}
		logger.Info().Msg("added finalizer to PaasConfig")
	}

	if config.GetDeletionTimestamp() != nil {
		logger.Info().Msg("paasconfig marked for deletion")
		// TODO(portly-halicore-76) We don't allow deletions for now
		return ctrl.Result{}, nil
	}

	configList := &v1alpha1.PaasConfigList{}
	if err := pcr.List(ctx, configList); err != nil {
		return ctrl.Result{}, err
	}

	// Enforce singleton pattern
	if len(configList.Items) > 1 {
		pcr.Log.Error(fmt.Errorf("singleton violation"), "more than one PaasConfig instance found")
		// TODO(hikarukin) delete extra PaasConfig instances or just log the error and skip reconciliation?
		// status unknown
		return ctrl.Result{}, nil
	}

	// Don't need to check if configuration has changed because we use predicate
	pcr.Log.Info("configuration has changed, verifying and updating operator settings")

	if err := config.Verify(); err != nil {
		pcr.Log.Info("invalid PaasConfig, not updating", "PaasConfig", err.Error())
		// Als die active was, dan zet je hem nog actief maar met fouten
		// en de opmerking dat de vorige versie van de resource in feite de actieve is
		return ctrl.Result{}, nil
	}

	// Update the shared configuration store
	SetConfig(*config)
	pcr.Log.Info("updated shared PaasConfig", "PaasConfig", config.Spec)

	// Apply the new configuration dynamically
	pcr.applyConfiguration(_cnf.currentConfig)

	// Paas & PaasNs reconciliation is triggered by a Watch on PaasConfig

	return ctrl.Result{}, nil
}

func (r *PaasConfigReconciler) applyConfiguration(spec v1alpha1.PaasConfig) {
	// TODO add various application functions
}

// // Apply log level dynamically
// func (r *PaasConfigReconciler) setLogLevel(level string) {
//     switch level {
//     case "debug":
//         r.Log = ctrl.Log.WithName("debug")
//     case "info":
//         r.Log = ctrl.Log.WithName("info")
//     default:
//         r.Log = ctrl.Log.WithName("default")
//     }
// }

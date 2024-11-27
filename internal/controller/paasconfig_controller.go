/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const paasconfigFinalizer = "paasconfig.cpet.belastingdienst.nl/finalizer"

// PaasReconciler reconciles a Paas object
type PaasConfigReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	Log               logr.Logger
	currentPaasConfig v1alpha1.PaasConfigSpec
	configMutex       sync.Mutex // For thread-safe updates
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

// func (pr PaasConfigReconciler) GetScheme() *runtime.Scheme {
// 	return pr.Scheme
// }

func (pcr *PaasConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pcr.Log.Info("reconciling PaasConfig")

	// Fetch all instances of PaasConfig
	var configList v1alpha1.PaasConfigList
	if err := pcr.List(ctx, &configList, &client.ListOptions{}); err != nil {
		return ctrl.Result{}, err
	}

	// Enforce singleton pattern
	if len(configList.Items) > 1 {
		pcr.Log.Error(fmt.Errorf("singleton violation"), "more than one PaasConfig instance found")
		// TODO delete extra PaasConfig instances or just log the error and skip reconciliation?
		return ctrl.Result{}, nil
	}

	// Fetch the singleton PaasConfig instance
	var config v1alpha1.PaasConfig
	// TODO use hardcoded namespacedname or something else?
	if err := pcr.Get(ctx, types.NamespacedName{Name: "paas-system"}, &config); err != nil {
		if errors.IsNotFound(err) {
			// TODO PaasConfig instance not found, create a default one?
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Don't need to check if configuration has changed because we use predicate
	pcr.Log.Info("configuration has changed, updating operator settings")

	// Update the shared configuration store
	SetConfig(config)
	pcr.Log.Info("updated shared PaasConfig", "PaasConfig", config.Spec)

	// Apply the new configuration dynamically
	pcr.applyConfiguration(_cnf.currentConfig)

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

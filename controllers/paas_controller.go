/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mydomainv1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const paasFinalizer = "paas.cpet.belastingdienst.nl/finalizer"

// PaasReconciler reconciles a Paas object
type PaasReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paas/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Paas object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *PaasReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO(user): your logic here
	paas := &mydomainv1alpha1.Paas{}
	logger := getLogger(ctx, paas, "PaaS", req.NamespacedName.String())
	logger.Info("Reconciling the PAAS object " + req.NamespacedName.String())

	err := r.Get(context.TODO(), req.NamespacedName, paas)
	if err != nil {
		if errors.IsNotFound(err) {
			// Something fishy is going on
			// Maybe someone cleaned the finalizers and then removed the PaaS project?
			logger.Error(err, req.NamespacedName.Name+" is already gone")
			return ctrl.Result{}, fmt.Errorf("PaaS object %s already gone", req.NamespacedName)
		}
		return ctrl.Result{}, nil
	} else if paas.GetDeletionTimestamp() != nil {
		logger.Info("PAAS object marked for deletion" + req.NamespacedName.String())
		if controllerutil.ContainsFinalizer(paas, paasFinalizer) {
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizePaaS(ctx, paas); err != nil {
				return ctrl.Result{}, err
			}

			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(paas, paasFinalizer)
			if err := r.Update(ctx, paas); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	logger.Info("Adding finalizer for PaaS object" + req.NamespacedName.String())
	if !controllerutil.ContainsFinalizer(paas, paasFinalizer) {
		logger.Info("PaaS object has no finalizer yet" + req.NamespacedName.String())
		controllerutil.AddFinalizer(paas, paasFinalizer)
		logger.Info("Added finalizer for PaaS object" + req.NamespacedName.String())
		if err := r.Update(ctx, paas); err != nil {
			logger.Info("Error updating PaaS object" + req.NamespacedName.String())
			logger.Info(fmt.Sprintf("%v", paas))
			return ctrl.Result{}, err
		}
		logger.Info("Updated PaaS object" + req.NamespacedName.String())
	}

	logger.Info("Creating quotas for PAAS object " + req.NamespacedName.String())
	// Create quotas if needed
	for _, q := range r.backendQuotas(ctx, paas) {
		logger.Info("Creating quota " + q.Name + " for PAAS object " + req.NamespacedName.String())
		if err := r.EnsureQuota(ctx, req, q); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating quota %s", q.ObjectMeta.Name))
			return ctrl.Result{}, err
		}
	}

	logger.Info("Creating namespaces for PAAS object " + req.NamespacedName.String())
	// Create namespaces if needed
	for _, ns := range r.backendNamespaces(ctx, paas) {
		if err := r.EnsureNamespace(req, ns); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating namespace %s", ns.ObjectMeta.Name))
			return ctrl.Result{}, err
		}
	}

	logger.Info("Extending Applicationsets for PAAS object" + req.NamespacedName.String())
	if err := r.EnsureAppSetCaps(ctx, paas); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Creating groups for PAAS object " + req.NamespacedName.String())
	for _, group := range r.backendGroups(ctx, paas) {
		if err := r.EnsureGroup(group); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating group %s", group.ObjectMeta.Name))
			return ctrl.Result{}, err
		}
	}

	logger.Info("Creating ldap groups for PAAS object " + req.NamespacedName.String())
	if err := r.EnsureLdapGroups(ctx, paas); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PaasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mydomainv1alpha1.Paas{}).
		Complete(r)
}

func (r *PaasReconciler) finalizePaaS(ctx context.Context, paas *mydomainv1alpha1.Paas) error {
	logger := getLogger(ctx, paas, "PaaS", ".finalizer")
	logger.Info("Finalizing PaaS")
	if err := r.FinalizeAppSetCaps(ctx, paas); err != nil {
		return err
	} else if err = r.FinalizeClusterQuotas(ctx, paas); err != nil {
		return err
	} else if cleanedLdapQueries, err := r.FinalizeGroups(ctx, paas); err != nil {
		// The whole idea is that groups (which are resources)
		// can also be ldapGroups (lines in a field in a configmap)
		// ldapGroups are only cleaned if the corresponding group is also cleaned
		logger.Error(err, "Cleanup of groups")
		if ldapErr := r.FinalizeLdapGroups(ctx, paas, cleanedLdapQueries); err != nil {
			logger.Error(ldapErr, "Cleanup of ldap groups")
		}
		return err
	} else if err = r.FinalizeLdapGroups(ctx, paas, cleanedLdapQueries); err != nil {
		logger.Error(err, "Cleanup of ldap groups")
		return err
	}
	logger.Info("Successfully finalized PaaS")
	return nil
}

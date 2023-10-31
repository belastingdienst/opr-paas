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

	"time"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
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

//+kubebuilder:rbac:groups=quota.openshift.io,resources=clusterresourcequotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=user.openshift.io,resources=groups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=argoproj.io,resources=argocds;applicationsets;applications;appprojects,verbs=create;delete
//+kubebuilder:rbac:groups=,resources=secrets;configmaps;namespaces,verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=create;delete;get;list;patch;update;watch;escallate
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=bind,resourceNames=admin

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
	paas := &v1alpha1.Paas{}
	logger := getLogger(ctx, paas, "PaaS", req.Name)
	logger.Info("Reconciling the PAAS object")

	err := r.Get(ctx, req.NamespacedName, paas)
	if err != nil {
		if errors.IsNotFound(err) {
			// Something fishy is going on
			// Maybe someone cleaned the finalizers and then removed the PaaS project?
			logger.Info(req.NamespacedName.Name + " is already gone")
			//return ctrl.Result{}, fmt.Errorf("PaaS object %s already gone", req.NamespacedName)
		}
		return ctrl.Result{}, nil
	} else if paas.GetDeletionTimestamp() != nil {
		logger.Info("PAAS object marked for deletion")
		if controllerutil.ContainsFinalizer(paas, paasFinalizer) {
			logger.Info("Finalizing PaaS")
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizePaaS(ctx, paas); err != nil {
				return ctrl.Result{}, err
			}

			logger.Info("Removing finalizer")
			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(paas, paasFinalizer)
			if err := r.Update(ctx, paas); err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("Finalization finished")
		}
		return ctrl.Result{}, nil
	}

	paas.Status.Truncate()
	defer r.Status().Update(ctx, paas)

	// Add finalizer for this CR
	logger.Info("Adding finalizer for PaaS object")
	if !controllerutil.ContainsFinalizer(paas, paasFinalizer) {
		logger.Info("PaaS object has no finalizer yet")
		controllerutil.AddFinalizer(paas, paasFinalizer)
		logger.Info("Added finalizer for PaaS object")
		if err := r.Update(ctx, paas); err != nil {
			logger.Info("Error updating PaaS object")
			logger.Info(fmt.Sprintf("%v", paas))
			return ctrl.Result{}, err
		}
		logger.Info("Updated PaaS object")
	}

	logger.Info("Creating quotas for PAAS object ")
	// Create quotas if needed
	for _, q := range r.BackendEnabledQuotas(ctx, paas) {
		logger.Info("Creating quota " + q.Name + " for PAAS object ")
		if err := r.EnsureQuota(ctx, paas, req, q); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating quota %s", q.ObjectMeta.Name))
			return ctrl.Result{}, err
		}
	}
	for _, name := range r.BackendDisabledQuotas(ctx, paas) {
		logger.Info("Cleaning quota " + name + " for PAAS object ")
		if err := r.FinalizeClusterQuota(ctx, paas, name); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating quota %s", name))
			return ctrl.Result{}, err
		}
	}

	logger.Info("Creating paas namespaces")
	for name := range paas.PrefixedAllEnabledNamespaces() {
		groups := paas.Spec.Groups.Names()
		secrets := paas.GetNsSshSecrets(name)
		pns := r.GetPaasNs(name, groups, secrets)
		if err := r.EnsurePaasNs(ctx, paas, req, pns); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating PaasNS %s", pns.ObjectMeta.Name))
			return ctrl.Result{}, err
		}
	}

	logger.Info("Creating Argo App for client bootstrapping")
	// Create bootstrap Argo App
	if paas.Spec.Capabilities.ArgoCD.Enabled {
		if err := r.EnsureArgoApp(ctx, paas); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.FinalizeArgoApp(ctx, paas); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Creating Ssh secrets")
	// Create argo ssh secrets
	secrets := r.BackendSecrets(ctx, paas)
	for _, secret := range secrets {
		if err = r.EnsureSecret(ctx, paas, secret); err != nil {
			logger.Error(err, "Failure while creating secret", "secret", secret)
			return ctrl.Result{}, err
		}
		logger.Info("Ssh secret succesfully created", "secret", secret)
	}

	logger.Info("Creating Argo Project")
	if err := r.EnsureAppProject(ctx, paas); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Extending Applicationsets for PAAS object")
	if err := r.EnsureAppSetCaps(ctx, paas); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Creating groups for PAAS object ")
	for _, group := range r.BackendGroups(ctx, paas) {
		if err := r.EnsureGroup(ctx, paas, group); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating group %s", group.ObjectMeta.Name))
			return ctrl.Result{}, err
		}
	}

	logger.Info("Creating ldap groups for PAAS object ")
	if err := r.EnsureLdapGroups(ctx, paas); err != nil {
		return ctrl.Result{}, err
	}

	if paas.Spec.Capabilities.ArgoCD.Enabled {
		retries := int(getConfig().ArgoPermissions.Retries)
		for i := 1; i <= retries; i++ {
			logger.Info(fmt.Sprintf("Updating ArgoCD Permissions (try %d/%d)", i, retries))
			if err := r.EnsureArgoPermissions(ctx, paas); err != nil {
				if i == retries {
					logger.Error(err, "updating ArgoCD Permissions failed", "retries", retries)
					return ctrl.Result{}, fmt.Errorf("updating ArgoCD Permissions failed %d times", retries)
				}
				time.Sleep(time.Second)
			} else {
				break
			}
		}
	}

	if err = r.ReconcileExtraClusterRoleBindings(ctx, paas); err != nil {
		logger.Error(err, "Reconciling Extra ClusterRoleBindings failed")
		return ctrl.Result{}, fmt.Errorf("reconciling Extra ClusterRoleBindings failed")
	}

	logger.Info("Updating PaaS object status")
	paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusReconcile, paas, "succeeded")
	logger.Info("PAAS object succesfully reconciled")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PaasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Paas{}).
		Complete(r)
}

func (r *PaasReconciler) finalizePaaS(ctx context.Context, paas *v1alpha1.Paas) error {
	logger := getLogger(ctx, paas, "PaaS", "finalizer code")
	logger.Info("Inside PaaS finalizer")
	if err := r.FinalizeAppSetCaps(ctx, paas); err != nil {
		logger.Error(err, "AppSet finalizer error")
		return err
	} else if err = r.FinalizeClusterQuotas(ctx, paas); err != nil {
		logger.Error(err, "Quota finalizer error")
		return err
	} else if cleanedLdapQueries, err := r.FinalizeGroups(ctx, paas); err != nil {
		// The whole idea is that groups (which are resources)
		// can also be ldapGroups (lines in a field in a configmap)
		// ldapGroups are only cleaned if the corresponding group is also cleaned
		logger.Error(err, "Group finalizer error")
		if ldapErr := r.FinalizeLdapGroups(ctx, paas, cleanedLdapQueries); err != nil {
			logger.Error(ldapErr, "And ldapGroup finalizer error")
		}
		return err
	} else if err = r.FinalizeLdapGroups(ctx, paas, cleanedLdapQueries); err != nil {
		logger.Error(err, "LdapGroup finalizer error")
		return err
	} else if err = r.FinalizeArgoApp(ctx, paas); err != nil {
		logger.Error(err, "ArgoApp finalizer error")
		return err
	} else if r.FinalizeExtraClusterRoleBindings(ctx, paas); err != nil {
		logger.Error(err, "Extra ClusterRoleBindings finalizer error")
		return err
	}
	logger.Info("PaaS succesfully finalized")
	return nil
}

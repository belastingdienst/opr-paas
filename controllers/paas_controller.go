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
	"sigs.k8s.io/controller-runtime/pkg/log"

	mydomainv1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
)

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
	log := log.FromContext(ctx).WithValues("PaaS", req.NamespacedName)
	log.Info("Reconciling the PAAS object " + req.NamespacedName.String())

	// TODO(user): your logic here
	paas := &mydomainv1alpha1.Paas{}

	err := r.Get(context.TODO(), req.NamespacedName, paas)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("PAAS object " + req.NamespacedName.Name + " is already gone")
			return ctrl.Result{}, r.cleanClusterQuotas(ctx, req.NamespacedName.String())
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	} else if paas.GetDeletionTimestamp() != nil {
		log.Info("PAAS object " + paas.Name + " is being deleted")
		return ctrl.Result{}, r.cleanClusterQuotas(ctx, req.NamespacedName.String())
	}

	log.Info("Creating quotas for PAAS object " + req.NamespacedName.String())
	// Create quotas if needed
	for _, q := range r.backendQuotas(paas) {
		log.Info("Creating quota " + q.Name + " for PAAS object " + req.NamespacedName.String())
		if err := r.ensureQuota(req, q); err != nil {
			log.Error(err, fmt.Sprintf("Failure while creating quota %s", q.ObjectMeta.Name))
			return ctrl.Result{}, err
		}
	}

	log.Info("Creating namespaces for PAAS object " + req.NamespacedName.String())
	// Create namespaces if needed
	for _, ns := range r.backendNamespaces(paas) {
		if err := r.ensureNamespace(req, paas, ns); err != nil {
			log.Error(err, fmt.Sprintf("Failure while creating namespace %s", ns.ObjectMeta.Name))
			return ctrl.Result{}, err
		}
	}

	log.Info("Extending Applicationsets for PAAS object" + req.NamespacedName.String())
	if err = r.ensureAppSetCaps(paas); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Creating groups for PAAS object " + req.NamespacedName.String())
	for _, group := range r.backendGroups(paas) {
		if err := r.ensureGroup(group); err != nil {
			log.Error(err, fmt.Sprintf("Failure while creating group %s", group.ObjectMeta.Name))
			return ctrl.Result{}, err
		}
	}

	log.Info("Creating ldap groups for PAAS object " + req.NamespacedName.String())
	if err = r.EnsureLdapGroups(paas); err != nil {
		return ctrl.Result{}, err
	}

	// Deployment and Service already exists - don't requeue
	// log.Info("Skip reconcile: Deployment and service already exists",
	// 	"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PaasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mydomainv1alpha1.Paas{}).
		Complete(r)
}

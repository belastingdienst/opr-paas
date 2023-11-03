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
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
)

const paasNsFinalizer = "paasns.cpet.belastingdienst.nl/finalizer"

// PaasNSReconciler reconciles a PaasNS object
type PaasNSReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PaasNS object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *PaasNSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	paasns := &v1alpha1.PaasNS{}
	var paas *v1alpha1.Paas
	logger := getLogger(ctx, paasns, paasns.Kind, req.Name)
	logger.Info("Reconciling the PaasNs object")

	if err := r.Get(ctx, req.NamespacedName, paasns); err != nil {
		if errors.IsNotFound(err) {
			// Something fishy is going on
			// Maybe someone cleaned the finalizers and then removed the PaasNs resource?
			logger.Info(req.NamespacedName.Name + " is already gone")
			//return ctrl.Result{}, fmt.Errorf("PaaS object %s already gone", req.NamespacedName)
		}
		return ctrl.Result{}, nil
	} else if paasns.GetDeletionTimestamp() != nil {
		logger.Info("PaasNS object marked for deletion")
		if controllerutil.ContainsFinalizer(paasns, paasNsFinalizer) {
			logger.Info("Finalizing PaasNs")
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizePaasNs(ctx, paasns); err != nil {
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
	} else if paas, err = r.paasFromPaasNs(ctx, paasns); err != nil {
		logger.Error(err, fmt.Sprintf("Cannot find PaaS %s", paasns.Spec.Paas))
		return ctrl.Result{}, err
	} else if paas == nil {
		err = fmt.Errorf("how can PaaS %s be %v here?", paasns.Spec.Paas, paas)
		logger.Error(err, "This is bad")
		return ctrl.Result{}, err
	} else if !paas.AmIOwner(paasns.OwnerReferences) {
		paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, paas, "updating owner")
		controllerutil.SetControllerReference(paas, paasns, r.Scheme)
	}

	nsName := paasns.NamespaceName()
	nsQuota := paas.Name
	if _, exists := paas.Spec.Capabilities.AsMap()[paasns.Name]; exists {
		nsQuota = nsName
	}

	if ns, err := BackendNamespace(ctx, paas, nsName, nsQuota, r.Scheme); err != nil {
		logger.Error(err, fmt.Sprintf("Failure while defining namespace %s", nsName))
		return ctrl.Result{}, err
	} else if err := EnsureNamespace(r.Client, ctx, paas, req, ns, r.Scheme); err != nil {
		logger.Error(err, fmt.Sprintf("Failure while creating namespace %s", ns.ObjectMeta.Name))
		return ctrl.Result{}, err
	}

	logger.Info("Creating paas-admin RoleBinding for PAASNS object")
	groupKeys := intersect(paas.Spec.Groups.Names(), paasns.Spec.Groups)
	rb := r.backendAdminRoleBinding(ctx, paas, types.NamespacedName{Namespace: nsName, Name: "paas-admin"}, groupKeys)
	if err := r.EnsureAdminRoleBinding(ctx, paas, rb); err != nil {
		logger.Error(err, fmt.Sprintf("Failure while creating rolebinding %s/%s",
			rb.ObjectMeta.Namespace,
			rb.ObjectMeta.Name))
		return ctrl.Result{}, err
	}

	logger.Info("Creating Ssh secrets")
	// Create argo ssh secrets
	secrets := r.BackendSecrets(ctx, paasns, paas)
	for _, secret := range secrets {
		if err := r.EnsureSecret(ctx, paas, secret); err != nil {
			logger.Error(err, "Failure while creating secret", "secret", secret)
			return ctrl.Result{}, err
		}
		logger.Info("Ssh secret succesfully created", "secret", secret)
	}
	/*
		- Create Extrapermissions
		- labels (voor makkelijk terugzoeken)
	*/

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PaasNSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PaasNS{}).
		Complete(r)
}

// nsFromNs gets all PaasNs objects from a namespace and returns a list of all the corresponding namespaces
// It also returns PaasNS in those namespaces recursively.
func (r *PaasNSReconciler) nssFromNs(ctx context.Context, ns string) map[string]bool {
	nss := make(map[string]bool)
	pnsList := &v1alpha1.PaasNSList{}
	if err := r.List(ctx, pnsList, &client.ListOptions{Namespace: ns}); err != nil {
		return nss
	}
	for _, pns := range pnsList.Items {
		nss[pns.NamespaceName()] = true
		// Call myself (recursiveness)
		for key := range r.nssFromNs(ctx, pns.NamespaceName()) {
			nss[key] = true
		}
	}
	return nss
}

// nsFromPaas accepts a PaaS and returns a list of all namespaces managed by this PaaS
// nsFromPaas uses nsFromNs which is recursive.
func (r *PaasNSReconciler) nssFromPaas(ctx context.Context, paas *v1alpha1.Paas) map[string]bool {
	// all nss to start with is all ns from paas, and ns named after paas
	sourceNss := paas.PrefixedAllEnabledNamespaces()
	sourceNss[paas.Name] = true
	// now scan them and append then to the end result
	finalNss := make(map[string]bool)
	for ns := range sourceNss {
		finalNss[ns] = true
		for key := range r.nssFromNs(ctx, ns) {
			finalNss[key] = true
		}
	}
	return finalNss
}

func (r *PaasNSReconciler) paasFromPaasNs(ctx context.Context, paasns *v1alpha1.PaasNS) (paas *v1alpha1.Paas, err error) {
	logger := getLogger(ctx, paasns, "PaasNs", "paasFromPaasNs")
	paas = &v1alpha1.Paas{}
	if err := r.Get(ctx, types.NamespacedName{Name: paasns.Spec.Paas}, paas); err != nil {
		logger.Error(err, fmt.Sprintf("Cannot find PaaS %s", paasns.Spec.Paas))
		return nil, err
	} else if paas.Name == "" {
		logger.Error(fmt.Errorf("PaaS %v is empty", paasns.Spec.Paas), "Why was an empty PaaS returned?")
		return nil, err
	}
	if paasns.Namespace == paas.Name {
		logger.Info(fmt.Sprintf("PaaS is %v", *paas))
		return paas, nil
	} else {
		namespaces := r.nssFromPaas(ctx, paas)
		logger.Info(fmt.Sprintf("PaaS namespaces are %v", namespaces))
		if _, exists := namespaces[paasns.Namespace]; exists {
			logger.Info(fmt.Sprintf("PaaS is %v", *paas))
			return paas, nil
		} else {
			var nss []string
			for key := range namespaces {
				nss = append(nss, key)
			}
			logger.Error(err, fmt.Sprintf(
				"PaasNs %s claims to come from paas %s, but %s is not in the list of namespaces coming from %s (%s)",
				types.NamespacedName{Name: paasns.Name, Namespace: paasns.Namespace},
				paas.Name,
				paasns.Namespace,
				paas.Name,
				strings.Join(nss, ", ")))
			return nil, err
		}
	}
}

func (r *PaasNSReconciler) finalizePaasNs(ctx context.Context, paasns *v1alpha1.PaasNS) error {
	logger := getLogger(ctx, paasns, "PaasNs", "finalizePaasNs")

	paas, err := r.paasFromPaasNs(ctx, paasns)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Cannot find PaaS %s", paasns.Spec.Paas))
		return err
	}

	logger.Info("Inside PaasNs finalizer")
	if err := r.FinalizeNamespace(ctx, paasns, paas); err != nil {
		logger.Error(err, fmt.Sprintf("Cannot remove namespace belonging to PaaS %s", paasns.Spec.Paas))
		return err
	}
	logger.Info("PaaS succesfully finalized")
	return nil
}

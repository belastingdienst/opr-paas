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

	var err error

	paasns := &v1alpha1.PaasNS{}
	var paas *v1alpha1.Paas
	logger := getLogger(ctx, paasns, paasns.Kind, req.Name)
	logger.Info("Reconciling the PaasNs object")

	if err = r.Get(ctx, req.NamespacedName, paasns); err != nil {
		if errors.IsNotFound(err) {
			// Something fishy is going on
			// Maybe someone cleaned the finalizers and then removed the PaasNs resource?
			logger.Info(req.NamespacedName.Name + " is already gone")
			//return ctrl.Result{}, fmt.Errorf("PaasNs object %s already gone", req.NamespacedName)
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
			controllerutil.RemoveFinalizer(paasns, paasNsFinalizer)
			if err := r.Update(ctx, paasns); err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("Finalization finished")
		}
		return ctrl.Result{}, nil
	}

	paasns.Status.Truncate()
	defer r.Status().Update(ctx, paasns)

	// Add finalizer for this CR
	logger.Info("Adding finalizer for PaasNs object")
	if !controllerutil.ContainsFinalizer(paasns, paasNsFinalizer) {
		logger.Info("PaasNs  object has no finalizer yet")
		controllerutil.AddFinalizer(paasns, paasNsFinalizer)
		logger.Info("Added finalizer for PaasNs  object")
		if err := r.Update(ctx, paasns); err != nil {
			logger.Info("Error updating PaasNs object")
			logger.Info(fmt.Sprintf("%v", paasns))
			return ctrl.Result{}, err
		}
		logger.Info("Updated PaasNs object")
	}

	if paas, _, err = r.paasFromPaasNs(ctx, paasns); err != nil {
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
		// This cannot be resolved by itself, so we should not have this keep on reconciling
		return ctrl.Result{}, nil
	} else if paas == nil {
		err = fmt.Errorf("how can PaaS %s be %v here?", paasns.Spec.Paas, paas)
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
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
		err = fmt.Errorf("failure while defining namespace %s: %s", nsName, err.Error())
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
		return ctrl.Result{}, err
	} else if err := EnsureNamespace(r.Client, ctx, paasns.Status.AddMessage, paas, req, ns, r.Scheme); err != nil {
		err = fmt.Errorf("failure while creating namespace %s: %s", nsName, err.Error())
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, ns, err.Error())
		return ctrl.Result{}, err
	}

	logger.Info("Creating paas-admin RoleBinding for PAASNS object")
	groupKeys := intersect(paas.Spec.Groups.Names(), paasns.Spec.Groups)
	rb := r.backendAdminRoleBinding(ctx, paas, types.NamespacedName{Namespace: nsName, Name: "paas-admin"}, groupKeys)
	if err := r.EnsureAdminRoleBinding(ctx, paas, rb); err != nil {
		err = fmt.Errorf("failure while creating rolebinding %s/%s: %s", rb.ObjectMeta.Namespace, rb.ObjectMeta.Name, err.Error())
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, rb, err.Error())
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

	if err = r.ReconcileExtraClusterRoleBinding(ctx, paasns, paas); err != nil {
		logger.Error(err, "Reconciling Extra ClusterRoleBindings failed")
		return ctrl.Result{}, fmt.Errorf("reconciling Extra ClusterRoleBindings failed")
	}

	if _, exists := paas.Spec.Capabilities.AsMap()[paasns.Name]; exists {
		if paasns.Name == "argocd" {
			logger.Info("Creating Argo App for client bootstrapping")
			// Create bootstrap Argo App
			if err := r.EnsureArgoApp(ctx, paasns, paas); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.EnsureArgoCD(ctx, paasns); err != nil {
				return ctrl.Result{}, err
			}
		}

		logger.Info("Extending Applicationsets for PAAS object")
		if err := r.EnsureAppSetCap(ctx, paasns, paas); err != nil {
			return ctrl.Result{}, err
		}
	}

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
func (r *PaasNSReconciler) nssFromNs(ctx context.Context, ns string) map[string]int {
	nss := make(map[string]int)
	pnsList := &v1alpha1.PaasNSList{}
	if err := r.List(ctx, pnsList, &client.ListOptions{Namespace: ns}); err != nil {
		return nss
	}
	for _, pns := range pnsList.Items {
		nsName := pns.NamespaceName()
		if value, exists := nss[nsName]; exists {
			nss[nsName] = value + 1
			// Call myself (recursiveness)
			for key, value := range r.nssFromNs(ctx, nsName) {
				nss[key] += value
			}
		} else {
			nss[nsName] = 1
		}
	}
	return nss
}

// nsFromPaas accepts a PaaS and returns a list of all namespaces managed by this PaaS
// nsFromPaas uses nsFromNs which is recursive.
func (r *PaasNSReconciler) nssFromPaas(ctx context.Context, paas *v1alpha1.Paas) map[string]int {
	// all nss to start with is all ns from paas, and ns named after paas
	sourceNss := paas.PrefixedAllEnabledNamespaces()
	sourceNss[paas.Name] = true
	// now scan them and append then to the end result
	finalNss := make(map[string]int)
	for ns := range sourceNss {
		finalNss[ns] = 1
		for key, value := range r.nssFromNs(ctx, ns) {
			finalNss[key] += value
		}
	}
	return finalNss
}

func (r *PaasNSReconciler) paasFromPaasNs(ctx context.Context, paasns *v1alpha1.PaasNS) (paas *v1alpha1.Paas, namespaces map[string]int, err error) {
	logger := getLogger(ctx, paasns, "PaasNs", "paasFromPaasNs")
	paas = &v1alpha1.Paas{}
	if err := r.Get(ctx, types.NamespacedName{Name: paasns.Spec.Paas}, paas); err != nil {
		err = fmt.Errorf("cannot find PaaS %s", paasns.Spec.Paas)
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
		logger.Error(err, fmt.Sprintf("Cannot find PaaS %s", paasns.Spec.Paas))
		return nil, namespaces, err
	} else if paas.Name == "" {
		err = fmt.Errorf("PaaS %v is empty", paasns.Spec.Paas)
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
		return nil, namespaces, err
	}
	if paasns.Namespace == paas.Name {
		return paas, r.nssFromPaas(ctx, paas), nil
	} else {
		namespaces = r.nssFromPaas(ctx, paas)
		if _, exists := namespaces[paasns.Namespace]; exists {
			return paas, namespaces, nil
		} else {
			var nss []string
			for key := range namespaces {
				nss = append(nss, key)
			}
			err = fmt.Errorf(
				"PaasNs %s claims to come from paas %s, but %s is not in the list of namespaces coming from %s (%s)",
				types.NamespacedName{Name: paasns.Name, Namespace: paasns.Namespace},
				paas.Name,
				paasns.Namespace,
				paas.Name,
				strings.Join(nss, ", "))
			paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
			return nil, map[string]int{}, err
		}
	}
}

func (r *PaasNSReconciler) finalizePaasNs(ctx context.Context, paasns *v1alpha1.PaasNS) error {
	logger := getLogger(ctx, paasns, "PaasNs", "finalizePaasNs")

	paas, nss, err := r.paasFromPaasNs(ctx, paasns)
	if err != nil {
		err = fmt.Errorf("cannot find PaaS %s: %s", paasns.Spec.Paas, err.Error())
		logger.Info(err.Error())
		return nil
	} else if nss[paasns.NamespaceName()] > 1 {
		err = fmt.Errorf("this is not the only paasns managing this namespace, silently removing this paasns")
		logger.Info(err.Error())
		return nil
	}

	logger.Info("Inside PaasNs finalizer")
	if err := r.FinalizeNamespace(ctx, paasns, paas); err != nil {
		err = fmt.Errorf("cannot remove namespace belonging to PaaS %s: %s", paasns.Spec.Paas, err.Error())
		return err
	}
	logger.Info("PaasNs succesfully finalized")
	return nil
}

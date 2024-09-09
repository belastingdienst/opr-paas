/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	v1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
)

const paasNsFinalizer = "paasns.cpet.belastingdienst.nl/finalizer"

// PaasNSReconciler reconciles a PaasNS object
type PaasNSReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (pr PaasNSReconciler) GetScheme() *runtime.Scheme {
	return pr.Scheme
}

//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cpet.belastingdienst.nl,resources=paasns/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the PaasNS object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//

func (r *PaasNSReconciler) GetPaasNs(ctx context.Context, req ctrl.Request) (paasns *v1alpha1.PaasNS, err error) {
	paasns = &v1alpha1.PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
		},
	}
	logger := getLogger(ctx, paasns, paasns.Kind, req.Name)
	logger.Info("Reconciling the PaasNs object")

	if err = r.Get(ctx, req.NamespacedName, paasns); err != nil {
		if errors.IsNotFound(err) {
			// Something fishy is going on
			// Maybe someone cleaned the finalizers and then removed the PaasNs resource?
			logger.Info(req.NamespacedName.Name + " is already gone")
			//return ctrl.Result{}, fmt.Errorf("PaasNs object %s already gone", req.NamespacedName)
			return nil, nil
		}
		return nil, err
	} else if paasns.GetDeletionTimestamp() != nil {
		logger.Info("PaasNS object marked for deletion")
		if controllerutil.ContainsFinalizer(paasns, paasNsFinalizer) {
			logger.Info("Finalizing PaasNs")
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizePaasNs(ctx, paasns); err != nil {
				return nil, err
			}

			logger.Info("Removing finalizer")
			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(paasns, paasNsFinalizer)
			if err := r.Update(ctx, paasns); err != nil {
				return nil, err
			}
			logger.Info("Finalization finished")
		}
		return nil, nil
	}

	// Add finalizer for this CR
	logger.Info("Adding finalizer for PaasNs object")
	if !controllerutil.ContainsFinalizer(paasns, paasNsFinalizer) {
		logger.Info("PaasNs  object has no finalizer yet")
		controllerutil.AddFinalizer(paasns, paasNsFinalizer)
		logger.Info("Added finalizer for PaasNs  object")
		if err := r.Update(ctx, paasns); err != nil {
			logger.Info("Error updating PaasNs object")
			logger.Info(fmt.Sprintf("%v", paasns))
			return nil, err
		}
		logger.Info("Updated PaasNs object")
	}
	return
}

func (r *PaasNSReconciler) GetPaas(ctx context.Context, paasns *v1alpha1.PaasNS) (paas *v1alpha1.Paas, err error) {
	if paas, _, err = r.paasFromPaasNs(ctx, paasns); err != nil {
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
		// This cannot be resolved by itself, so we should not have this keep on reconciling
		return nil, nil
	} else if paas == nil {
		err = fmt.Errorf("how can PaaS %s be %v here?", paasns.Spec.Paas, paas)
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
		return nil, err
	} else if !paas.AmIOwner(paasns.OwnerReferences) {
		paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, paas, "updating owner")
		controllerutil.SetControllerReference(paas, paasns, r.Scheme)
	}
	return
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *PaasNSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {

	paasns := &v1alpha1.PaasNS{ObjectMeta: metav1.ObjectMeta{Name: req.Name}}
	logger := getLogger(ctx, paasns, "PaasNs", req.Name)
	logger.Info("Reconciling the PaasNs object")

	if paasns, err = r.GetPaasNs(ctx, req); err != nil {
		logger.Error(err, "could not get PaaS from k8s")
		return
	} else if paasns == nil {
		logger.Error(err, "nothing to do")
		return
	}

	paasns.Status.Truncate()
	defer func() {
		logger.Info("Updating PaasNs status", "messages", len(paasns.Status.Messages))
		if err = r.Status().Update(ctx, paasns); err != nil {
			logger.Error(err, "Updating PaasNs status failed")
		}
	}()

	var paas *v1alpha1.Paas
	if paas, err = r.GetPaas(ctx, paasns); err != nil || paas == nil {
		// This cannot be resolved by itself, so we should not have this keep on reconciling
		return ctrl.Result{}, nil
	} else if r.ReconcileNamespaces(ctx, paas, paasns); err != nil {
		return ctrl.Result{}, err
	} else if r.ReconcileRolebindings(ctx, paas, paasns, logger); err != nil {
		return ctrl.Result{}, err
	} else if r.ReconcileSecrets(ctx, paas, paasns, logger); err != nil {
		return ctrl.Result{}, err
	} else if err = r.ReconcileExtraClusterRoleBinding(ctx, paasns, paas); err != nil {
		logger.Error(err, "Reconciling Extra ClusterRoleBindings failed")
		return ctrl.Result{}, fmt.Errorf("reconciling Extra ClusterRoleBindings failed")
	} else if _, exists := paas.Spec.Capabilities.AsMap()[paasns.Name]; exists {
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
		WithEventFilter(
			predicate.Or(
				// Spec updated
				predicate.GenerationChangedPredicate{},
				// Labels updated
				predicate.LabelChangedPredicate{},
			)).
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
		if value, exists := nss[nsName]; !exists {
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

// nsFromNs gets all PaasNs objects from a namespace and returns a list of all the corresponding namespaces
// It also returns PaasNS in those namespaces recursively.
func (r *PaasReconciler) pnsFromNs(ctx context.Context, ns string) map[string]v1alpha1.PaasNS {
	nss := make(map[string]v1alpha1.PaasNS)
	pnsList := &v1alpha1.PaasNSList{}
	if err := r.List(ctx, pnsList, &client.ListOptions{Namespace: ns}); err != nil {
		return nss
	}
	for _, pns := range pnsList.Items {
		nsName := pns.NamespaceName()
		if _, exists := nss[nsName]; !exists {
			nss[nsName] = pns
			// Call myself (recursiveness)
			for key, value := range r.pnsFromNs(ctx, nsName) {
				nss[key] = value
			}
		} else {
			nss[nsName] = pns
		}
	}
	return nss
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
	} else if err = r.finalizeAppSetCap(ctx, paasns); err != nil {
		err = fmt.Errorf("cannot remove paas from capability ApplicationSet belonging to PaaS %s: %s", paasns.Spec.Paas, err.Error())
		return err
	}
	if _, isCapability := paas.Spec.Capabilities.AsMap()[paasns.Name]; isCapability {
		logger.Info("PaasNs is a capability, also finalizing Cluster Resource Quota")
		if err := r.FinalizeClusterQuota(ctx, paasns); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while finalizing quota %s", paasns.Name))
			return err
		}
	}
	logger.Info("PaasNs succesfully finalized")
	return nil
}

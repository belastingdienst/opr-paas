/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	paasns = &v1alpha1.PaasNS{}
	ctx = setLogComponent(ctx, "paasns")
	logger := log.Ctx(ctx)
	logger.Info().Msg("reconciling PaasNs")

	if err = r.Get(ctx, req.NamespacedName, paasns); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(paasns, paasNsFinalizer) {
		if ok := controllerutil.AddFinalizer(paasns, paasNsFinalizer); !ok {
			return nil, fmt.Errorf("failed to add finalizer")
		}
		if err := r.Update(ctx, paasns); err != nil {
			logger.Err(err).Msg("error updating PaasNs")
			return nil, err
		}
		logger.Info().Msg("added finalizer to PaasNs")
	}

	// TODO(portly-halicore-76) Move to admission webhook once available
	// check if Config is set, as reconciling and finalizing without config, leaves object in limbo.
	// this is only an issue when object is being removed, finalizers will not be removed causing the object to be in limbo.
	if reflect.DeepEqual(v1alpha1.PaasConfigSpec{}, GetConfig()) {
		logger.Error().Msg("no config found")
		paasns.Status.Truncate()
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusReconcile, paasns, "please reach out to your system administrator as there is no Paasconfig available to reconcile against.")
		err := r.Status().Update(ctx, paasns)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no config found")
	}

	if paasns.GetDeletionTimestamp() != nil {
		logger.Info().Msg("paasNS object marked for deletion")
		if controllerutil.ContainsFinalizer(paasns, paasNsFinalizer) {
			logger.Info().Msg("finalizing PaasNs")
			// Run finalization logic for paasNsFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizePaasNs(ctx, paasns); err != nil {
				return nil, err
			}

			logger.Info().Msg("removing finalizer")
			// Remove paasNsFinalizer. Once all finalizers have been removed, the object will be deleted.
			controllerutil.RemoveFinalizer(paasns, paasNsFinalizer)
			if err := r.Update(ctx, paasns); err != nil {
				return nil, err
			}
			logger.Info().Msg("finalization finished")
		}
		return nil, nil
	}
	return paasns, nil
}

func (r *PaasNSReconciler) GetPaas(ctx context.Context, paasns *v1alpha1.PaasNS) (paas *v1alpha1.Paas, err error) {
	paas, _, err = r.paasFromPaasNs(ctx, paasns)
	if err != nil {
		if errors.IsNotFound(err) {
			err = fmt.Errorf("cannot find PaaS %s", paasns.Spec.Paas)
			paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
		}
		// This cannot be resolved by itself, so we should not have this keep on reconciling
		return nil, err
	}
	if !paas.AmIOwner(paasns.OwnerReferences) {
		paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, paas, "updating owner")
		if err := controllerutil.SetControllerReference(paas, paasns, r.Scheme); err != nil {
			return nil, err
		}
	}
	return
}

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (r *PaasNSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	paasns := &v1alpha1.PaasNS{ObjectMeta: metav1.ObjectMeta{Name: req.Name}}
	ctx, logger := setRequestLogger(ctx, paasns, r.Scheme, req)

	errResult := reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Second * 10,
	}
	okResult := reconcile.Result{
		Requeue: false,
	}

	//TODO(portly-halicore-76) do check if Config is set, else return and Requeue after say... minutes / hours ...
	// as reconciling and finalizing without config, causes operator in meh state.
	// this is only flacky when object is being removed, finalizers will not be removed
	if reflect.DeepEqual(v1alpha1.PaasConfigSpec{}, GetConfig()) {
		logger.Error().Msg("No config found")
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	if paasns, err = r.GetPaasNs(ctx, req); err != nil {
		// TODO(portly-halicore-76) move to admission webhook once available
		// Don't requeue that often
		if strings.Contains(err.Error(), "no config found") {
			return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
		}
		logger.Err(err).Msg("could not get PaasNs from k8s")
		return errResult, err
	}

	if paasns == nil {
		// r.GetPaasNs handled all logic and returned a nil object
		return okResult, nil
	}

	paasns.Status.Truncate()
	defer func() {
		logger.Info().
			Int("messages", len(paasns.Status.Messages)).
			Msg("updating PaasNs status")

		if err = r.Status().Update(ctx, paasns); err != nil {
			logger.Err(err).Msg("updating PaasNs status failed")
		}
	}()

	var paas *v1alpha1.Paas
	if paas, err = r.GetPaas(ctx, paasns); err != nil {
		// This cannot be resolved by itself, so we should not have this keep on reconciling
		return okResult, nil
	}

	err = r.ReconcileNamespaces(ctx, paas, paasns)
	if err != nil {
		return errResult, err
	}

	err = r.ReconcileRolebindings(ctx, paas, paasns)
	if err != nil {
		return errResult, err
	}

	err = r.ReconcileSecrets(ctx, paas, paasns)
	if err != nil {
		// error related to decrypting secret. User error, must not retry reconciliation
		if strings.Contains(err.Error(), "failed to decrypt secret") {
			return okResult, nil
		}
		return errResult, err
	}

	err = r.ReconcileExtraClusterRoleBinding(ctx, paasns, paas)
	if err != nil {
		logger.Err(err).Msg("reconciling Extra ClusterRoleBindings failed")
		return errResult, fmt.Errorf("reconciling Extra ClusterRoleBindings failed")
	}

	if _, exists := paas.Spec.Capabilities[paasns.Name]; exists {
		if paasns.Name == "argocd" {
			logger.Info().Msg("creating Argo App for client bootstrapping")

			// Create bootstrap Argo App
			if err := r.EnsureArgoApp(ctx, paasns, paas); err != nil {
				return errResult, err
			}

			if err := r.EnsureArgoCD(ctx, paasns); err != nil {
				return errResult, err
			}
		}

		logger.Info().Msg("extending Applicationsets for Paas object")
		if err := r.EnsureAppSetCap(ctx, paasns, paas); err != nil {
			return errResult, err
		}
	}

	logger.Info().Msg("updating PaasNs object status")
	paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusReconcile, paasns, "succeeded")
	logger.Info().Msg("paasNs object successfully reconciled")

	return okResult, nil
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

// nssFromNs gets all PaasNs objects from a namespace and returns a list of all the corresponding namespaces
// It also returns PaasNS in those namespaces recursively.
func (r *PaasNSReconciler) nssFromNs(ctx context.Context, ns string) map[string]int {
	nss := make(map[string]int)
	pnsList := &v1alpha1.PaasNSList{}
	if err := r.List(ctx, pnsList, &client.ListOptions{Namespace: ns}); err != nil {
		// In this case panic is ok, since this situation can only occur when either k8s is down, or permissions are insufficient.
		// Both cases we should not continue executing code...
		panic(err)
	}
	for _, pns := range pnsList.Items {
		nsName := pns.NamespaceName()
		if value, exists := nss[nsName]; exists {
			nss[nsName] = value + 1
		} else {
			nss[nsName] = 1
		}
		// Call myself (recursively)
		for key, value := range r.nssFromNs(ctx, nsName) {
			nss[key] += value
		}
	}
	return nss
}

// nsFromPaas accepts a PaaS and returns a list of all namespaces managed by this PaaS
// nsFromPaas uses nsFromNs which is recursive.
func (r *PaasNSReconciler) nssFromPaas(ctx context.Context, paas *v1alpha1.Paas) map[string]int {
	finalNss := make(map[string]int)
	finalNss[paas.Name] = 1
	for key, value := range r.nssFromNs(ctx, paas.Name) {
		finalNss[key] += value
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
	ctx = setLogComponent(ctx, "paasns")
	logger := log.Ctx(ctx)
	paas = &v1alpha1.Paas{}
	if err := r.Get(ctx, types.NamespacedName{Name: paasns.Spec.Paas}, paas); err != nil {
		logger.Err(err).Msg("cannot get Paas")
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
	ctx = setLogComponent(ctx, "paasns")
	logger := log.Ctx(ctx)

	paas, nss, err := r.paasFromPaasNs(ctx, paasns)
	if err != nil {
		err = fmt.Errorf("cannot find PaaS %s: %s", paasns.Spec.Paas, err.Error())
		logger.Info().Msg(err.Error())
		return nil
	} else if nss[paasns.NamespaceName()] > 1 {
		err = fmt.Errorf("this is not the only paasns managing this namespace, silently removing this paasns")
		logger.Info().Msg(err.Error())
		return nil
	}

	logger.Info().Msg("inside PaasNs finalizer")
	if err := r.FinalizeNamespace(ctx, paasns, paas); err != nil {
		err = fmt.Errorf("cannot remove namespace belonging to PaaS %s: %s", paasns.Spec.Paas, err.Error())
		return err
	} else if err = r.finalizeAppSetCap(ctx, paasns); err != nil {
		err = fmt.Errorf("cannot remove paas from capability ApplicationSet belonging to PaaS %s: %s", paasns.Spec.Paas, err.Error())
		return err
	}
	if _, isCapability := paas.Spec.Capabilities[paasns.Name]; isCapability {
		logger.Info().Msg("paasNs is a capability, also finalizing Cluster Resource Quota")
		if err := r.FinalizeClusterQuota(ctx, paasns); err != nil {
			logger.Err(err).Msg(fmt.Sprintf("Failure while finalizing quota %s", paasns.Name))
			return err
		}
	}
	logger.Info().Msg("paasNs successfully finalized")
	return nil
}

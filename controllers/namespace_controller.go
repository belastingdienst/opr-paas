package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ensureNamespace ensures Namespace presence in given namespace.
func (r *PaasReconciler) EnsureNamespace(
	ctx context.Context,
	paas *v1alpha1.Paas,
	request reconcile.Request,
	ns *corev1.Namespace,
) error {

	// See if namespace exists and create if it doesn't
	found := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{
		Name: ns.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the namespace
		err = r.Create(ctx, ns)

		if err != nil {
			// creating the namespace failed
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, ns, err.Error())
			return err
		} else {
			// creating the namespace was successful
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, ns, "succeeded")
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, ns, err.Error())
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "updating owner")
		controllerutil.SetControllerReference(paas, found, r.Scheme)
		return r.Update(ctx, found)
	}
	return nil

}

// backendNamespace is a code for Creating Namespace
func (r *PaasReconciler) backendNamespace(
	ctx context.Context,
	paas *v1alpha1.Paas,
	name string,
	quota string,
) *corev1.Namespace {
	logger := getLogger(ctx, paas, "Namespace", name)
	logger.Info(fmt.Sprintf("Defining %s Namespace", name))
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: paas.ClonedLabels(),
		},
		Spec: corev1.NamespaceSpec{},
	}
	logger.Info(fmt.Sprintf("Setting Quotagroup %s", quota))
	ns.ObjectMeta.Labels[getConfig().QuotaLabel] = quota

	argoNameSpace := fmt.Sprintf("%s-argocd", paas.Name)
	if paas.Spec.Capabilities.ArgoCD.Enabled && name != argoNameSpace {
		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate,
			ns, "Setting managed_by_label")
		ns.ObjectMeta.Labels[getConfig().ManagedByLabel] = argoNameSpace
	}
	paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate,
		ns, "Setting oplosgroep_label")
	ns.ObjectMeta.Labels[getConfig().OplosgroepLabel] = paas.Spec.Oplosgroep

	logger.Info("Setting Owner")
	controllerutil.SetControllerReference(paas, ns, r.Scheme)
	return ns
}

func (r *PaasReconciler) BackendEnabledNamespaces(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (ns []*corev1.Namespace) {

	for cap_name, cap := range paas.Spec.Capabilities.AsMap() {
		if cap.IsEnabled() {
			name := fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, cap_name)
			ns = append(ns, r.backendNamespace(ctx, paas, name, name))
		}
	}
	capNs := paas.AllCapNamespaces()
	for _, ns_suffix := range paas.Spec.Namespaces {
		name := fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, ns_suffix)
		n := r.backendNamespace(ctx, paas, name, paas.ObjectMeta.Name)
		if _, isCapNs := capNs[name]; isCapNs {
			paas.Status.AddMessage(v1alpha1.PaasStatusWarning, v1alpha1.PaasStatusCreate, n,
				"Skipping extra namespace, as it is also a capability namespace")
		} else {
			ns = append(ns, n)
		}
	}
	return ns
}

func (r *PaasReconciler) BackendDisabledNamespaces(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (ns []string) {
	for name, cap := range paas.Spec.Capabilities.AsMap() {
		if !cap.IsEnabled() {
			ns = append(ns, fmt.Sprintf("%s-%s", paas.Name, name))
		}
	}
	return ns
}

func (r *PaasReconciler) FinalizeNamespaces(ctx context.Context, paas *v1alpha1.Paas) error {
	logger := getLogger(ctx, paas, "Namespace", "")
	logger.Info("Finalizing")

	enabledNs := paas.AllEnabledNamespaces()

	// Loop through all namespaces and remove when not should be
	nsList := &corev1.NamespaceList{}
	if err := r.List(ctx, nsList); err != nil {
		return err
	}

	for _, ns := range nsList.Items {
		if !strings.HasPrefix(ns.Name, paas.Name+"-") {
			// logger.Info("Skipping finalization", "Namespace", ns.Name, "Reason", "wrong prefix")
		} else if !paas.AmIOwner(ns.OwnerReferences) {
			// logger.Info("Skipping finalization", "Namespace", ns.Name, "Reason", "I am not owner")
		} else if _, isEnabled := enabledNs[ns.Name]; isEnabled {
			// logger.Info("Skipping finalization", "Namespace", ns.Name, "Reason", "Should be there")
		} else if err := r.Delete(ctx, &ns); err != nil {
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusDelete, &ns, err.Error())
			// logger.Error(err, "Could not delete ns", "Namespace", ns.Name)
		} else {
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusDelete, &ns, "succeeded")
		}
	}
	return nil
}

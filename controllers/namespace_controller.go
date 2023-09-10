package controllers

import (
	"context"
	"fmt"

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
			paas.Status.AddMessage("ERROR", "create", ns.TypeMeta.String(), ns.Name, err.Error())
			return err
		} else {
			// creating the namespace was successful
			paas.Status.AddMessage("INFO", "create", ns.TypeMeta.String(), ns.Name, "succeeded")
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		paas.Status.AddMessage("ERROR", "find", ns.TypeMeta.String(), ns.Name, err.Error())
		return err
	}
	paas.Status.AddMessage("INFO", "find", ns.TypeMeta.String(), ns.Name, "already existed")

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
	//matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}
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
	for _, ns_suffix := range paas.Spec.Namespaces {
		name := fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, ns_suffix)
		ns = append(ns, r.backendNamespace(ctx, paas, name, paas.ObjectMeta.Name))
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

func (r *PaasReconciler) FinalizeNamespace(ctx context.Context, paas *v1alpha1.Paas, namespaceName string) error {
	logger := getLogger(ctx, paas, "Namespace", namespaceName)
	logger.Info("Finalizing")
	obj := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{
		Name: namespaceName,
	}, obj); err != nil && errors.IsNotFound(err) {
		logger.Info("Does not exist")
		return nil
	} else if err != nil {
		logger.Info("Error retrieving info: " + err.Error())
		return err
	} else {
		logger.Info("Deleting")
		return r.Delete(ctx, obj)
	}
}

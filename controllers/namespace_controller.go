package controllers

import (
	"context"
	"fmt"

	mydomainv1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ensureNamespace ensures Namespace presence in given namespace.
func (r *PaasReconciler) EnsureNamespace(
	request reconcile.Request,
	ns *corev1.Namespace,
) error {

	// See if namespace exists and create if it doesn't
	found := &corev1.Namespace{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Name: ns.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the namespace
		err = r.Create(context.TODO(), ns)

		if err != nil {
			// creating the namespace failed
			return err
		} else {
			// creating the namespace was successful
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		return err
	}

	return nil
}

// backendNamespace is a code for Creating Namespace
func (r *PaasReconciler) backendNamespace(
	ctx context.Context,
	paas *mydomainv1alpha1.Paas,
	suffix string,
) *corev1.Namespace {
	name := fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, suffix)
	logger := getLogger(ctx, paas, "Namespace", name)
	logger.Info(fmt.Sprintf("Defining %s Namepsace", name))
	//matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: paas.Labels,
		},
		Spec: corev1.NamespaceSpec{},
	}
	logger.Info(fmt.Sprintf("Setting Quotagroup %s", name))
	ns.ObjectMeta.Labels["clusterquotagroup"] = name

	logger.Info("Setting Owner")
	controllerutil.SetControllerReference(paas, ns, r.Scheme)
	return ns
}

func (r *PaasReconciler) backendNamespaces(
	ctx context.Context,
	paas *mydomainv1alpha1.Paas,
) (ns []*corev1.Namespace) {
	if paas.Spec.Capabilities.ArgoCD.Enabled {
		ns = append(ns, r.backendNamespace(ctx, paas, "argocd"))
	}
	if paas.Spec.Capabilities.CI.Enabled {
		ns = append(ns, r.backendNamespace(ctx, paas, "ci"))
	}
	if paas.Spec.Capabilities.Grafana.Enabled {
		ns = append(ns, r.backendNamespace(ctx, paas, "grafana"))
	}
	if paas.Spec.Capabilities.SSO.Enabled {
		ns = append(ns, r.backendNamespace(ctx, paas, "sso"))
	}
	return ns
}

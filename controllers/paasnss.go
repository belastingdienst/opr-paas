package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *PaasReconciler) GetPaasNs(ctx context.Context, paas *v1alpha1.Paas, name string,
	groups []string, secrets map[string]string, seperateQuota bool) *v1alpha1.PaasNS {
	pns := &v1alpha1.PaasNS{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PaasNS",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			// For now hardcoded, rethink when needed
			Namespace: fmt.Sprintf("%s-argocd", paas.Name),
			Labels:    paas.ClonedLabels(),
		},
		Spec: v1alpha1.PaasNSSpec{
			Groups:        groups,
			SshSecrets:    secrets,
			SeparateQuota: seperateQuota,
		},
	}
	logger := getLogger(ctx, paas, pns.Kind, name)
	logger.Info("Defining")
	paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate,
		pns, "Setting oplosgroep_label")
	pns.ObjectMeta.Labels[getConfig().OplosgroepLabel] = paas.Spec.Oplosgroep

	logger.Info("Setting Owner")
	controllerutil.SetControllerReference(paas, pns, r.Scheme)
	return pns
}

func (r *PaasReconciler) EnsurePaasNs(ctx context.Context, paas *v1alpha1.Paas, request reconcile.Request, pns *v1alpha1.PaasNS) error {
	logger := getLogger(ctx, paas, pns.Kind, pns.Name)
	logger.Info("Ensuring")

	// See if namespace exists and create if it doesn't
	found := &v1alpha1.PaasNS{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      pns.Name,
		Namespace: pns.Namespace,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		if err = r.Create(ctx, pns); err != nil {
			// creating the namespace failed
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, pns, err.Error())
			return err
		} else {
			// creating the namespace was successful
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, pns, "succeeded")
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, pns, err.Error())
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "updating owner")
		controllerutil.SetControllerReference(paas, found, r.Scheme)
	}
	var changed bool
	for key, value := range pns.ObjectMeta.Labels {
		if orgValue, exists := found.ObjectMeta.Labels[key]; !exists {
			// Not set yet
		} else if orgValue != value {
			// different
		} else {
			// No action required
			continue
		}
		changed = true
		found.ObjectMeta.Labels[key] = value
	}
	if changed {
		return r.Update(ctx, found)
	}
	return nil
}

package controllers

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	argocd "github.com/argoproj-labs/argocd-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ensureLdapGroup ensures Group presence
func (r *PaasReconciler) EnsureArgoPermissions(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	logger := getLogger(ctx, paas, "ArgoPermissions", "")
	// See if group already exists and create if it doesn't
	argo := &argocd.ArgoCD{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getConfig().ArgoPermissions.ResourceName,
			Namespace: fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, "argocd"),
		},
	}
	argoName := types.NamespacedName{
		Namespace: fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, "argocd"),
		Name:      getConfig().ArgoPermissions.ResourceName,
	}

	err := r.Get(ctx, argoName, argo)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("ArgoObject not found yet")
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, argo, err.Error())
		return fmt.Errorf("ArgoObject not found yet")
	} else if err != nil {
		logger.Error(err, "Could not retrieve ArgoCD")
		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusFind, argo, err.Error())
		return err
	}
	var oldPolicy string
	if argo.Spec.RBAC.Policy != nil {
		oldPolicy = *argo.Spec.RBAC.Policy
	}
	policy := getConfig().ArgoPermissions.FromGroups(
		paas.Spec.Groups.AsGroups().Keys())
	scopes := "[groups]"
	logger.Info(fmt.Sprintf("Setting ArgoCD permissions to %s", policy))
	if oldPolicy == policy {
		logger.Info("No policy changes")
		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, argo, "no policy changes")
		return nil
	}
	argo.Spec.RBAC.Policy = &policy
	argo.Spec.RBAC.Scopes = &scopes
	logger.Info("Updating ArgoCD object")
	paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, argo, "succeeded")
	paas.Status.ArgoCDUrl = argo.Status.Host
	return r.Update(ctx, argo)
}

/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	argocd "github.com/belastingdienst/opr-paas/internal/stubs/argocd/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ensureLdapGroup ensures Group presence
func (r *PaasNSReconciler) EnsureArgoCD(
	ctx context.Context,
	paasns *v1alpha1.PaasNS,
) error {
	if paasns.Name != "argocd" {
		return nil
	}
	logger := getLogger(ctx, paasns, "ArgoPermissions", "")

	policy := getConfig().ArgoPermissions.FromGroups(
		paasns.Spec.Groups)
	scopes := "[groups]"

	argo := &argocd.ArgoCD{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getConfig().ArgoPermissions.ResourceName,
			Namespace: paasns.NamespaceName(),
		},
		Spec: argocd.ArgoCDSpec{
			RBAC: argocd.ArgoCDRBACSpec{
				Policy: &policy,
				Scopes: &scopes,
			},
		},
	}

	argoName := types.NamespacedName{
		Namespace: paasns.NamespaceName(),
		Name:      getConfig().ArgoPermissions.ResourceName,
	}

	err := r.Get(ctx, argoName, argo)
	if err != nil && errors.IsNotFound(err) {
		paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, argo, "creating ArgoCD instance")
		return r.Create(ctx, argo)
	} else if err != nil {
		paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusFind, argo, err.Error())
		return err
	}
	var oldPolicy string
	if argo.Spec.RBAC.Policy != nil {
		oldPolicy = *argo.Spec.RBAC.Policy
	}
	logger.Info(fmt.Sprintf("Setting ArgoCD permissions to %s", policy))
	if oldPolicy == policy {
		paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, argo, "no policy changes")
		return nil
	}
	argo.Spec.RBAC.Policy = &policy
	argo.Spec.RBAC.Scopes = &scopes
	logger.Info("Updating ArgoCD object")
	paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, argo, "updating ArgoCD instance")
	return r.Update(ctx, argo)
}

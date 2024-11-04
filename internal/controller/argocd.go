/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argocd "github.com/belastingdienst/opr-paas/internal/stubs/argoproj-labs/v1beta1"
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
	paas, _, err := r.paasFromPaasNs(ctx, paasns)
	if err != nil {
		return err
	}
	logger := getLogger(ctx, paasns, "ArgoPermissions", "")

	defaultPolicy := getConfig().ArgoPermissions.DefaultPolicy
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
				DefaultPolicy: &defaultPolicy,
				Policy:        &policy,
				Scopes:        &scopes,
			},
		},
	}

	argoName := types.NamespacedName{
		Namespace: paasns.NamespaceName(),
		Name:      getConfig().ArgoPermissions.ResourceName,
	}

	err = controllerutil.SetControllerReference(paas, argo, r.Scheme)
	if err != nil {
		return err
	}

	err = r.Get(ctx, argoName, argo)
	if err != nil && errors.IsNotFound(err) {
		paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, argo, "creating ArgoCD instance")
		return r.Create(ctx, argo)
	} else if err != nil {
		paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusFind, argo, err.Error())
		return err
	}
	patch := client.MergeFrom(argo.DeepCopy())
	var oldPolicy string
	var oldDefaultPolicy string
	if argo.Spec.RBAC.Policy != nil {
		oldPolicy = *argo.Spec.RBAC.Policy
	}
	if argo.Spec.RBAC.DefaultPolicy != nil {
		oldDefaultPolicy = *argo.Spec.RBAC.DefaultPolicy
	}
	logger.Info(fmt.Sprintf("Setting ArgoCD permissions to %s", policy))
	if oldPolicy == policy && oldDefaultPolicy == defaultPolicy && paas.AmIOwner(argo.OwnerReferences) {
		paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, argo, "no changes")
		return nil
	}
	argo.Spec.RBAC.Policy = &policy
	argo.Spec.RBAC.Scopes = &scopes
	argo.Spec.RBAC.DefaultPolicy = &defaultPolicy
	if err = controllerutil.SetControllerReference(paas, argo, r.GetScheme()); err != nil {
		return err
	}
	logger.Info("Updating ArgoCD object")
	paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, argo, "updating ArgoCD instance")
	return r.Patch(ctx, argo, patch)
}

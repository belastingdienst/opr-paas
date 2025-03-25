/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"
	argocd "github.com/belastingdienst/opr-paas/internal/stubs/argoproj-labs/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EnsureArgoCD ensures ArgoCD instance
func (r *PaasReconciler) EnsureArgoCD(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	ctx, logger := logging.GetLogComponent(ctx, "argopermissions")

	namespace := fmt.Sprintf("%s-%s", paas.Name, "argocd")

	defaultPolicy := config.GetConfigSpec().ArgoPermissions.DefaultPolicy
	policy := config.GetConfigSpec().ArgoPermissions.FromGroups(paas.GroupNames())
	scopes := "[groups]"

	argo := &argocd.ArgoCD{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.GetConfigSpec().ArgoPermissions.ResourceName,
			Namespace: namespace,
		},
		Spec: argocd.ArgoCDSpec{
			RBAC: argocd.ArgoCDRBACSpec{
				DefaultPolicy: &defaultPolicy,
				Policy:        &policy,
				Scopes:        &scopes,
			},
		},
	}

	err := controllerutil.SetControllerReference(paas, argo, r.Scheme)
	if err != nil {
		return err
	}

	err = r.Get(ctx, types.NamespacedName{
		Namespace: argo.Namespace,
		Name:      argo.Name,
	}, argo)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, argo)
	} else if err != nil {
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
	logger.Info().Msgf("setting ArgoCD permissions to %s", policy)
	if oldPolicy == policy && oldDefaultPolicy == defaultPolicy && paas.AmIOwner(argo.OwnerReferences) {
		return nil
	}
	argo.Spec.RBAC.Policy = &policy
	argo.Spec.RBAC.Scopes = &scopes
	argo.Spec.RBAC.DefaultPolicy = &defaultPolicy
	if err = controllerutil.SetControllerReference(paas, argo, r.GetScheme()); err != nil {
		return err
	}
	logger.Info().Msg("updating ArgoCD object")
	return r.Patch(ctx, argo, patch)
}

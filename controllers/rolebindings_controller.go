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

	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureRoleBinding ensures RoleBinding presence in given rolebinding.
func (r *PaasNSReconciler) EnsureAdminRoleBinding(
	ctx context.Context,
	paas *v1alpha1.Paas,
	rb *rbac.RoleBinding,
) error {
	namespacedName := types.NamespacedName{
		Name:      rb.Name,
		Namespace: rb.Namespace,
	}
	// See if rolebinding exists and create if it doesn't
	found := &rbac.RoleBinding{}
	err := r.Get(ctx, namespacedName, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the rolebinding
		err = r.Create(ctx, rb)

		if err != nil {
			// creating the rolebinding failed
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, rb, err.Error())
			return err
		} else {
			// creating the rolebinding was successful
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, rb, "succeeded")
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the rolebinding not existing
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, rb, err.Error())
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "updating owner")
		controllerutil.SetControllerReference(paas, found, r.Scheme)
		return r.Update(ctx, found)
	}
	return nil

}

// backendRoleBinding is a code for Creating RoleBinding
func (r *PaasNSReconciler) backendAdminRoleBinding(
	ctx context.Context,
	paas *v1alpha1.Paas,
	name types.NamespacedName,
	groups []string,
) *rbac.RoleBinding {
	logger := getLogger(ctx, paas, "RoleBinding", name.String())
	logger.Info(fmt.Sprintf("Defining %s RoleBinding", name))

	var subjects = []rbac.Subject{}
	for _, g := range groups {
		subjects = append(subjects,
			rbac.Subject{
				Kind:     "Group",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     g,
			})
	}

	rb := &rbac.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    paas.ClonedLabels(),
		},
		Subjects: subjects,
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "admin",
		},
	}
	logger.Info("Setting Owner")
	controllerutil.SetControllerReference(paas, rb, r.Scheme)
	return rb
}

func (r *PaasNSReconciler) BackendEnabledRoleBindings(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (rb []*rbac.RoleBinding) {
	groupKeys := paas.Spec.Groups.Names()
	for ns_name := range paas.PrefixedAllEnabledNamespaces() {
		name := types.NamespacedName{
			Name:      "paas-admin",
			Namespace: ns_name,
		}
		rb = append(rb, r.backendAdminRoleBinding(ctx, paas, name, groupKeys))
	}
	return rb
}

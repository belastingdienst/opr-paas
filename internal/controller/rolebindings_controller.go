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
	"github.com/go-logr/logr"

	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func diffRbacSubjects(l1 []rbac.Subject, l2 []rbac.Subject) bool {
	subResults := make(map[string]bool)
	for _, s := range l1 {
		key := fmt.Sprintf("%s.%s.%s", s.Namespace, s.Name, s.Kind)
		subResults[key] = false
	}
	for _, s := range l2 {
		key := fmt.Sprintf("%s.%s.%s", s.Namespace, s.Name, s.Kind)
		if _, exists := subResults[key]; !exists {
			// Something is in l2, but not in l1
			return true
		} else {
			subResults[key] = true
		}
	}
	for _, value := range subResults {
		if !value {
			// Something is in l2, but not in l1
			return true
		}
	}
	return false
}

// ensureRoleBinding ensures RoleBinding presence in given rolebinding.
func EnsureRoleBinding(
	ctx context.Context,
	r Reconciler,
	paas *v1alpha1.Paas,
	statusMessages *v1alpha1.PaasNsStatus,
	rb *rbac.RoleBinding,
) error {
	logger := getLogger(ctx, paas, "Rolebinding", rb.Name)
	if len(rb.Subjects) < 1 {
		return FinalizeRoleBinding(ctx, r, statusMessages, rb)
	}
	namespacedName := types.NamespacedName{
		Name:      rb.Name,
		Namespace: rb.Namespace,
	}
	// See if rolebinding exists and create if it doesn't
	found := &rbac.RoleBinding{}
	err := r.Get(ctx, namespacedName, found)
	if err != nil && errors.IsNotFound(err) {
		// Create the rolebinding
		logger.Info("Creating RoleBinding", "Namespace", rb.Namespace, "Name", rb.Name, "roleRef", rb.RoleRef, "subject", rb.Subjects)
		err = r.Create(ctx, rb)

		if err != nil {
			// Creating the rolebinding failed
			logger.Error(err, "Error creating rolebinding")
			statusMessages.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, rb, err.Error())
			return err
		} else {
			// Creating the rolebinding was successful and return
			logger.Info("Created rolebinding")
			statusMessages.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, rb, "succeeded")
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the rolebinding not existing
		logger.Error(err, "Error getting rolebinding")
		statusMessages.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, rb, err.Error())
		return err
	}
	var changed bool
	if !paas.AmIOwner(found.OwnerReferences) {
		if err = controllerutil.SetControllerReference(paas, found, r.GetScheme()); err != nil {
			logger.Error(err, "Error setting rolebinding owner")
			return err
		}
		changed = true
	}
	if diffRbacSubjects(found.Subjects, rb.Subjects) {
		found.Subjects = rb.Subjects
		changed = true
	}
	if changed {
		logger.Info("Updating RoleBinding", "Namespace", rb.Namespace, "Name", rb.Name, "roleRef", rb.RoleRef, "subject", rb.Subjects)
		if err = r.Update(ctx, found); err != nil {
			logger.Error(err, "Error updating rolebinding")
			statusMessages.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusUpdate, rb, err.Error())
			return err
		}
		statusMessages.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "updated")
	} else {
		statusMessages.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "not needed")
	}
	return err
}

// backendRoleBinding is a code for Creating RoleBinding
func backendRoleBinding(
	ctx context.Context,
	r Reconciler,
	paas *v1alpha1.Paas,
	name types.NamespacedName,
	role string,
	groups []string,
) (*rbac.RoleBinding, error) {
	logger := getLogger(ctx, paas, "RoleBinding", name.String())
	logger.Info(fmt.Sprintf("Defining %s RoleBinding", name))

	subjects := []rbac.Subject{}
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
			APIVersion: "rbac.authorization.k8s.io/v1",
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
			Name:     role,
		},
	}
	logger.Info("Setting Owner")
	if err := controllerutil.SetControllerReference(paas, rb, r.GetScheme()); err != nil {
		return rb, err
	}

	return rb, nil
}

// ensureRoleBinding ensures RoleBinding presence in given rolebinding.
func FinalizeRoleBinding(
	ctx context.Context,
	r Reconciler,
	statusMessages *v1alpha1.PaasNsStatus,
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
		return nil
	} else if err != nil {
		// Error that isn't due to the rolebinding not existing
		statusMessages.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, rb, err.Error())
		return err
	} else {
		statusMessages.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusDelete, rb, "Succeeded")
		return r.Delete(ctx, rb)
	}
}

// ReconcileRolebindings is used by the Paas reconciler to reconcile RB's
func (r *PaasReconciler) ReconcileRolebindings(
	ctx context.Context,
	paas *v1alpha1.Paas,
	logger logr.Logger,
) error {
	for _, paasns := range r.pnsFromNs(ctx, paas.ObjectMeta.Name) {
		roles := make(map[string][]string)

		// Guarantee use of value for current iteration when referencing
		paasns := paasns
		for _, roleList := range getConfig().RoleMappings {
			for _, role := range roleList {
				roles[role] = []string{}
			}
		}
		logger.Info("All roles", "Rolebindings map", roles)
		for groupName, groupRoles := range paas.Spec.Groups.Filtered(paasns.Spec.Groups).Roles() {
			for _, mappedRole := range getConfig().RoleMappings.Roles(groupRoles) {
				if role, exists := roles[mappedRole]; exists {
					roles[mappedRole] = append(role, groupName)
				} else {
					roles[mappedRole] = []string{groupName}
				}
			}
		}
		logger.Info("Creating paas RoleBindings for PAASNS object", "Rolebindings map", roles)
		for roleName, groupKeys := range roles {
			statusMessages := v1alpha1.PaasNsStatus{}
			rbName := types.NamespacedName{Namespace: paasns.NamespaceName(), Name: fmt.Sprintf("paas-%s", roleName)}
			logger.Info("Creating Rolebinding", "role", roleName, "groups", groupKeys)
			rb, _ := backendRoleBinding(ctx, r, paas, rbName, roleName, groupKeys)
			if err := EnsureRoleBinding(ctx, r, paas, &statusMessages, rb); err != nil {
				err = fmt.Errorf("failure while creating/updating rolebinding %s/%s: %s", rb.ObjectMeta.Namespace, rb.ObjectMeta.Name, err.Error())
				paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, rb, err.Error())
				return err
			}
			paas.Status.AddMessages(statusMessages.GetMessages())
		}
	}
	return nil
}

// ReconcileRolebindings is used by the PaasNS reconciler to reconcile RB's
func (r *PaasNSReconciler) ReconcileRolebindings(
	ctx context.Context,
	paas *v1alpha1.Paas,
	paasns *v1alpha1.PaasNS,
	logger logr.Logger,
) error {
	// Creating a list of roles and the groups that should have them, for this namespace
	roles := make(map[string][]string)
	for groupName, groupRoles := range paas.Spec.Groups.Filtered(paasns.Spec.Groups).Roles() {
		for _, mappedRole := range getConfig().RoleMappings.Roles(groupRoles) {
			if role, exists := roles[mappedRole]; exists {
				roles[mappedRole] = append(role, groupName)
			} else {
				roles[mappedRole] = []string{groupName}
			}
		}
	}
	logger.Info("Creating paas RoleBindings for PAASNS object", "Rolebindings map", roles)
	for roleName, groupKeys := range roles {
		rbName := types.NamespacedName{Namespace: paasns.NamespaceName(), Name: fmt.Sprintf("paas-%s", roleName)}
		logger.Info("Creating Rolebinding", "role", roleName, "groups", groupKeys)
		rb, _ := backendRoleBinding(ctx, r, paas, rbName, roleName, groupKeys)
		if err := EnsureRoleBinding(ctx, r, paas, &paasns.Status, rb); err != nil {
			err = fmt.Errorf("failure while creating rolebinding %s/%s: %s", rb.ObjectMeta.Namespace, rb.ObjectMeta.Name, err.Error())
			paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, rb, err.Error())
			return err
		}
	}
	return nil
}

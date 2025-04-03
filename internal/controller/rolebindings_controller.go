/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"

	"github.com/rs/zerolog/log"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureRoleBinding ensures RoleBinding presence in given rolebinding.
func ensureRoleBinding(
	ctx context.Context,
	r Reconciler,
	paas *v1alpha1.Paas,
	rb *rbac.RoleBinding,
) error {
	logger := log.Ctx(ctx)
	if len(rb.Subjects) < 1 {
		return finalizeRoleBinding(ctx, r, rb)
	}
	namespacedName := types.NamespacedName{
		Name:      rb.Name,
		Namespace: rb.Namespace,
	}
	// See if rolebinding exists and create if it doesn't
	found := &rbac.RoleBinding{}
	err := r.Get(ctx, namespacedName, found)
	if err != nil && errors.IsNotFound(err) {
		return createRoleBinding(ctx, r, rb)
	} else if err != nil {
		// Error that isn't due to the rolebinding not existing
		logger.Err(err).Msg("error getting rolebinding")
		return err
	}
	var changed bool
	if !paas.AmIOwner(found.OwnerReferences) {
		if err = controllerutil.SetControllerReference(paas, found, r.GetScheme()); err != nil {
			logger.Err(err).Msg("error setting rolebinding owner")
			return err
		}
		changed = true
	}
	if !reflect.DeepEqual(found.Subjects, rb.Subjects) {
		found.Subjects = rb.Subjects
		changed = true
	}
	if changed {
		logger.Info().
			Str("Namespace", rb.Namespace).
			Str("Name", rb.Name).
			Str("roleRef", rb.RoleRef.Name).
			Any("subject", rb.Subjects).
			Msg("updating RoleBinding")
		if err = r.Update(ctx, found); err != nil {
			logger.Err(err).Msg("error updating rolebinding")
			return err
		}
	}
	return nil
}

func createRoleBinding(
	ctx context.Context,
	r Reconciler,
	rb *rbac.RoleBinding,
) error {
	logger := log.Ctx(ctx)
	// Create the rolebinding
	logger.Info().
		Str("Namespace", rb.Namespace).
		Str("Name", rb.Name).
		Str("roleRef", rb.RoleRef.Name).
		Any("subject", rb.Subjects).
		Msg("creating RoleBinding")
	err := r.Create(ctx, rb)
	if err != nil {
		// Creating the rolebinding failed
		logger.Err(err).Msg("error creating rolebinding")
		return err
	}

	// Creating the rolebinding was successful and return
	logger.Info().Msg("created rolebinding")
	return nil
}

// backendRoleBinding is code for defining RoleBindings
func backendRoleBinding(
	ctx context.Context,
	r Reconciler,
	paas *v1alpha1.Paas,
	name types.NamespacedName,
	role string,
	groupNames []string,
) (*rbac.RoleBinding, error) {
	logger := log.Ctx(ctx)
	logger.Info().Msgf("defining %s RoleBinding", name)
	subjects := []rbac.Subject{}
	for _, groupName := range groupNames {
		subjects = append(subjects,
			rbac.Subject{
				Kind:     "Group",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     groupName,
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
	logger.Info().Msg("setting Owner")
	if err := controllerutil.SetControllerReference(paas, rb, r.GetScheme()); err != nil {
		return rb, err
	}

	return rb, nil
}

// finalizeRoleBinding ensures RoleBinding presence in given rolebinding.
func finalizeRoleBinding(
	ctx context.Context,
	r Reconciler,
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
		return err
	} else {
		return r.Delete(ctx, rb)
	}
}

// reconcileRolebindings is used by the Paas reconciler to reconcile RB's
func (r *PaasReconciler) reconcileRolebindings(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	ctx, logger := logging.GetLogComponent(ctx, "rolebinding")
	for _, paasns := range r.pnsFromNs(ctx, paas.Name) {
		roles := make(map[string][]string)

		// Guarantee use of value for current iteration when referencing
		paasns := paasns
		for _, roleList := range config.GetConfig().Spec.RoleMappings {
			for _, role := range roleList {
				roles[role] = []string{}
			}
		}
		logger.Info().Any("Rolebindings map", roles).Msg("all roles")
		for groupKey, groupRoles := range paas.Spec.Groups.Filtered(paasns.Spec.Groups).Roles() {
			logger.Info().Msgf("defining Rolebindings for Group %s", groupKey)
			// Convert the groupKey to a groupName to map the rolebinding subjects to a group
			groupName := paas.GroupKey2GroupName(groupKey)
			for _, mappedRole := range config.GetConfig().Spec.RoleMappings.Roles(groupRoles) {
				if role, exists := roles[mappedRole]; exists {
					roles[mappedRole] = append(role, groupName)
				} else {
					roles[mappedRole] = []string{groupName}
				}
			}
		}
		logger.Info().Any("Rolebindings map", roles).Msg("creating paas RoleBindings for PAASNS object")
		for roleName, groupNames := range roles {
			rbName := types.NamespacedName{Namespace: paasns.NamespaceName(), Name: fmt.Sprintf("paas-%s", roleName)}
			logger.Debug().
				Str("role", roleName).
				Strs("groups", groupNames).
				Msg("creating Rolebinding")
			rb, err := backendRoleBinding(ctx, r, paas, rbName, roleName, groupNames)
			if err != nil {
				return err
			}
			if err := ensureRoleBinding(ctx, r, paas, rb); err != nil {
				err = fmt.Errorf(
					"failure while creating/updating rolebinding %s/%s: %s",
					rb.Namespace,
					rb.Name,
					err.Error(),
				)
				return err
			}
		}
	}
	return nil
}

// ReconcileRolebindings is used by the PaasNS reconciler to reconcile RB's
func (r *PaasNSReconciler) ReconcileRolebindings(
	ctx context.Context,
	paas *v1alpha1.Paas,
	paasns *v1alpha1.PaasNS,
) error {
	ctx, logger := logging.GetLogComponent(ctx, "rolebinding")
	// Creating a list of roles and the groups that should have them, for this namespace
	roles := make(map[string][]string)
	for groupKey, groupRoles := range paas.Spec.Groups.Filtered(paasns.Spec.Groups).Roles() {
		// Convert the groupKey to a groupName to map the rolebinding subjects to a group
		groupName := paas.GroupKey2GroupName(groupKey)
		for _, mappedRole := range config.GetConfig().Spec.RoleMappings.Roles(groupRoles) {
			if role, exists := roles[mappedRole]; exists {
				roles[mappedRole] = append(role, groupName)
			} else {
				roles[mappedRole] = []string{groupName}
			}
		}
	}
	logger.Info().
		Any("Rolebindings map", roles).
		Msg("creating paas RoleBindings for PaasNs object")
	for roleName, groupNames := range roles {
		rbName := types.NamespacedName{Namespace: paasns.NamespaceName(), Name: fmt.Sprintf("paas-%s", roleName)}
		logger.Debug().
			Str("role", roleName).
			Strs("groups", groupNames).
			Msg("creating Rolebinding")
		rb, err := backendRoleBinding(ctx, r, paas, rbName, roleName, groupNames)
		if err != nil {
			return err
		}
		if err := ensureRoleBinding(ctx, r, paas, rb); err != nil {
			err = fmt.Errorf("failure while creating rolebinding %s/%s: %s", rb.Namespace, rb.Name, err.Error())
			return err
		}
	}
	return nil
}

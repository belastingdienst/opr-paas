/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v4/internal/config"
	"github.com/belastingdienst/opr-paas/v4/internal/logging"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const crbNamePrefix string = "paas"

// TODO are these labels still correct?
var defaultCRBLabels = map[string]string{
	"app.kubernetes.io/created-by": "opr-paas",
	"app.kubernetes.io/part-of":    "opr-paas",
}

func (r *PaasReconciler) getClusterRoleBinding(
	ctx context.Context,
	role string,
) (crb *rbac.ClusterRoleBinding, err error) {
	crbName := join(crbNamePrefix, role)
	found := &rbac.ClusterRoleBinding{}
	err = r.Get(ctx, types.NamespacedName{Name: crbName}, found)
	if err != nil && errors.IsNotFound(err) {
		return backendClusterRoleBinding(role), nil
	} else if err != nil {
		return nil, err
	}
	return found, nil
}

func (r *PaasReconciler) getClusterRoleBindingsWithLabel(
	ctx context.Context,
	labelMatcher client.MatchingLabels,
) (crbs *rbac.ClusterRoleBindingList, err error) {
	list := &rbac.ClusterRoleBindingList{}

	err = r.List(ctx, list, labelMatcher)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *PaasReconciler) updateClusterRoleBinding(
	ctx context.Context,
	crb *rbac.ClusterRoleBinding,
) (err error) {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerClusterRoleBindingsComponent)
	if len(crb.Subjects) == 0 && crb.ResourceVersion != "" {
		logger.Info().Msgf("cleaning empty ClusterRoleBinding %s", crb.Name)
		return r.Delete(ctx, crb)
	} else if len(crb.Subjects) != 0 && crb.ResourceVersion == "" {
		logger.Info().Msgf("creating new ClusterRoleBinding %s", crb.Name)
		return r.Create(ctx, crb)
	} else if len(crb.Subjects) != 0 {
		logger.Info().Msgf("updating existing ClusterRoleBinding %s", crb.Name)
		return r.Update(ctx, crb)
	}
	return nil
}

func backendClusterRoleBinding(
	role string,
) *rbac.ClusterRoleBinding {
	crbName := join(crbNamePrefix, role)
	rb := &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   crbName,
			Labels: defaultCRBLabels,
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     role,
		},
	}
	return rb
}

func addSAToClusterRoleBinding(
	crb *rbac.ClusterRoleBinding,
	namespace string,
	sa string,
) (changed bool) {
	for _, subject := range crb.Subjects {
		if (subject.Kind == "ServiceAccount") && (subject.Namespace == namespace) && (subject.Name == sa) {
			// SA is already in this CRB
			return false
		}
	}
	// If it was not already in the list, lets add it
	crb.Subjects = append(crb.Subjects, rbac.Subject{
		Kind:      "ServiceAccount",
		Name:      sa,
		Namespace: namespace,
	})
	return true
}

func updateClusterRoleBindingForRemovedSA(
	crb *rbac.ClusterRoleBinding,
	nsRe regexp.Regexp,
	sa string,
) (changed bool) {
	var newSubjects []rbac.Subject

	for _, subject := range crb.Subjects {
		if nsRe.MatchString(subject.Namespace) && (subject.Kind == "ServiceAccount") &&
			(subject.Name == sa || sa == "") {
			// Subject is this sa, don't keep.
			changed = true
			continue
		}
		newSubjects = append(newSubjects, subject)
	}
	crb.Subjects = newSubjects
	return changed
}

func addOrUpdateCrb(
	ctx context.Context,
	crb *rbac.ClusterRoleBinding,
	nsName string,
	sas map[string]bool,
) (changed bool) {
	_, logger := logging.GetLogComponent(ctx, logging.ControllerClusterRoleBindingsComponent)
	crbName := crb.Name
	for sa, add := range sas {
		if add {
			if isAdded := addSAToClusterRoleBinding(crb, nsName, sa); isAdded {
				logger.Info().Msgf("adding sa %s for ns %s to crb %v", sa, nsName, crbName)
				changed = true
			}
			logger.Info().Msgf("sa %s in ns %s already added to crb %v", sa, nsName, crbName)
		} else {
			nsRe := *regexp.MustCompile(fmt.Sprintf("^%s$", nsName))
			if isRemoved := updateClusterRoleBindingForRemovedSA(crb, nsRe, sa); isRemoved {
				logger.Info().Msgf("deleting sa %s for ns %s from crb %s", sa, nsName, crbName)
				changed = true
			}
			logger.Info().Msgf("sa %s in ns %s no longer in crb %s", sa, nsName, crbName)
		}
	}
	return changed
}

func (r *PaasReconciler) reconcileClusterRoleBinding(
	ctx context.Context,
	paas *v1alpha2.Paas,
	nsName string,
	capName string,
) (err error) {
	var crb *rbac.ClusterRoleBinding
	capability, capExists := paas.Spec.Capabilities[capName]
	myConfig, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return err
	}
	capConfig, capConfigExists := myConfig.Spec.Capabilities[capName]
	if !capConfigExists && !capExists {
		return err
	}

	ctx, _ = logging.GetLogComponent(ctx, logging.ControllerClusterRoleBindingsComponent)
	permissions := capConfig.ExtraPermissions.AsConfigRolesSas(capability.ExtraPermissions)
	permissions.Merge(capConfig.DefaultPermissions.AsConfigRolesSas(true))
	for role, sas := range permissions {
		if crb, err = r.getClusterRoleBinding(ctx, role); err != nil {
			return err
		}
		if addOrUpdateCrb(ctx, crb, nsName, sas) {
			if err = r.updateClusterRoleBinding(ctx, crb); err != nil {
				return err
			}
		}
	}

	err = r.checkForRemovedPermissionsInPaasConfig(ctx, capConfig, permissions)
	if err != nil {
		return err
	}

	return nil
}

func (r *PaasReconciler) checkForRemovedPermissionsInPaasConfig(
	ctx context.Context,
	capConfig v1alpha2.ConfigCapability,
	permissions v1alpha2.ConfigRolesSas,
) error {
	_, logger := logging.GetLogComponent(ctx, logging.ControllerClusterRoleBindingsComponent)

	if len(capConfig.DefaultPermissions.AsConfigRolesSas(true)) == 0 {
		logger.Info().Msg("Default permissions for capability in PaasConfig are empty, " +
			"checking if any CRBs need to be removed")

		var crbs *rbac.ClusterRoleBindingList
		crbs, err := r.getClusterRoleBindingsWithLabel(ctx, defaultCRBLabels)
		if err != nil {
			return err
		}

		for _, crbFromList := range crbs.Items {
			keepItem := false
			for role := range permissions {
				expectedCrbName := crbNamePrefix + role
				if crbFromList.Name == expectedCrbName {
					keepItem = true
				}
			}
			if !keepItem {
				logger.Info().Msgf("Deleting ClusterRoleBinding with name %s as it is no longer present in PaasConfig",
					crbFromList.Name)
				err = r.Delete(ctx, &crbFromList)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *PaasReconciler) reconcileClusterRoleBindings(
	ctx context.Context,
	paas *v1alpha2.Paas,
	nsDefs namespaceDefs,
) (err error) {
	for _, nsDef := range nsDefs {
		err = r.reconcileClusterRoleBinding(ctx, paas, nsDef.nsName, nsDef.capName)
		if err != nil {
			return err
		}
	}
	return r.finalizeCapClusterRoleBindings(ctx, paas)
}

func subjectsFromCrb(crb rbac.ClusterRoleBinding) []string {
	var subjects []string
	for _, subject := range crb.Subjects {
		subjects = append(subjects, fmt.Sprintf("%s/%s/%s", subject.Kind, subject.Namespace, subject.Name))
	}
	return subjects
}

func (r *PaasReconciler) finalizeClusterRoleBinding(
	ctx context.Context,
	role string,
	nsRegularExpression regexp.Regexp,
) error {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerClusterRoleBindingsComponent)
	crb, err := r.getClusterRoleBinding(ctx, role)
	if err != nil {
		return err
	}
	logger.Info().Msgf("subjects before update: %s", strings.Join(subjectsFromCrb(*crb), ", "))
	changed := updateClusterRoleBindingForRemovedSA(crb,
		nsRegularExpression, "")
	logger.Info().Msgf("subjects after update: %s", strings.Join(subjectsFromCrb(*crb), ", "))
	if !changed {
		logger.Info().Msg("no changes")
		return nil
	}
	logger.Info().Msgf("updating rolebinding %s after cleaning SA's for '%s'", role, nsRegularExpression.String())
	return r.updateClusterRoleBinding(ctx, crb)
}

func (r *PaasReconciler) finalizeCapClusterRoleBindings(ctx context.Context, paas *v1alpha2.Paas) error {
	myConfig, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return err
	}
	for capName, capConfig := range myConfig.Spec.Capabilities {
		nsRE := regexp.MustCompile(fmt.Sprintf("^%s-%s$", paas.Name, capName))
		if _, isDefined := paas.Spec.Capabilities[capName]; isDefined {
			continue
		}
		var roles []string
		for _, defRoles := range capConfig.DefaultPermissions {
			roles = append(roles, defRoles...)
		}
		for _, extraRoles := range capConfig.ExtraPermissions {
			roles = append(roles, extraRoles...)
		}
		for _, role := range roles {
			err = r.finalizeClusterRoleBinding(ctx, role, *nsRE)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *PaasReconciler) finalizePaasClusterRoleBindings(
	ctx context.Context,
	paas *v1alpha2.Paas,
) (err error) {
	myConfig, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return err
	}
	var capRoles []string
	for _, capConfig := range myConfig.Spec.Capabilities {
		capRoles = append(capRoles, capConfig.ExtraPermissions.Roles()...)
		capRoles = append(capRoles, capConfig.DefaultPermissions.Roles()...)
	}
	for _, role := range capRoles {
		re := regexp.MustCompile(fmt.Sprintf("^%s-", paas.Name))
		err = r.finalizeClusterRoleBinding(ctx, role, *re)
		if err != nil {
			return err
		}
	}
	return nil
}

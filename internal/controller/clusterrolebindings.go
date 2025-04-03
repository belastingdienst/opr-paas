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

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"

	"github.com/rs/zerolog/log"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const crbNameFormat string = "paas-%s"

// getClusterRoleBinding returns a ClusterRoleBinding to enable extra permissions for certain capabilities.
func getClusterRoleBinding(
	ctx context.Context,
	r client.Client,
	role string,
) (crb *rbac.ClusterRoleBinding, err error) {
	crbName := fmt.Sprintf(crbNameFormat, role)
	found := &rbac.ClusterRoleBinding{}
	err = r.Get(ctx, types.NamespacedName{Name: crbName}, found)
	if err != nil && errors.IsNotFound(err) {
		return backendClusterRoleBinding(role), nil
	} else if err != nil {
		return nil, err
	} else {
		return found, nil
	}
}

func updateClusterRoleBinding(
	ctx context.Context,
	r client.Client,
	crb *rbac.ClusterRoleBinding,
) (err error) {
	logger := log.Ctx(ctx)
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

// backendClusterRoleBinding is a code for Creating RoleBinding
func backendClusterRoleBinding(
	role string,
) *rbac.ClusterRoleBinding {
	crbName := fmt.Sprintf(crbNameFormat, role)
	rb := &rbac.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: crbName,
			// TODO are these labels still correct?
			Labels: map[string]string{
				"app.kubernetes.io/created-by": "opr-paas",
				"app.kubernetes.io/part-of":    "opr-paas",
			},
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
	paasns *v1alpha1.PaasNS,
	crb *rbac.ClusterRoleBinding,
	sas map[string]bool,
) (changed bool) {
	logger := log.Ctx(ctx)
	crbName := crb.Name
	for sa, add := range sas {
		if add {
			if isAdded := addSAToClusterRoleBinding(crb, paasns.NamespaceName(), sa); isAdded {
				logger.Info().Msgf("adding sa %s for ns %s to crb %v", sa, paasns.NamespaceName(), crbName)
				changed = true
			}
			logger.Info().Msgf("sa %s in ns %s already added to crb %v", sa, paasns.NamespaceName(), crbName)
		} else {
			nsRe := *regexp.MustCompile(fmt.Sprintf("^%s$", paasns.NamespaceName()))
			if isRemoved := updateClusterRoleBindingForRemovedSA(crb, nsRe, sa); isRemoved {
				logger.Info().Msgf("deleting sa %s for ns %s from crb %s", sa, paasns.NamespaceName(), crbName)
				changed = true
			}
			logger.Info().Msgf("sa %s in ns %s no longer in crb %s", sa, paasns.NamespaceName(), crbName)
		}
	}
	return changed
}

func (r *PaasNSReconciler) ReconcileExtraClusterRoleBinding(
	ctx context.Context,
	paasns *v1alpha1.PaasNS,
	paas *v1alpha1.Paas,
) (err error) {
	var crb *rbac.ClusterRoleBinding
	capability, capExists := paas.Spec.Capabilities[paasns.Name]
	capConfig, capConfigExists := config.GetConfig().Spec.Capabilities[paasns.Name]
	if !capConfigExists && !capExists {
		return err
	}

	ctx, _ = logging.GetLogComponent(ctx, "clusterrolebinding")

	permissions := capConfig.ExtraPermissions.AsConfigRolesSas(capability.WithExtraPermissions())
	permissions.Merge(capConfig.DefaultPermissions.AsConfigRolesSas(true))
	for role, sas := range permissions {
		if crb, err = getClusterRoleBinding(ctx, r.Client, role); err != nil {
			return err
		}
		if addOrUpdateCrb(ctx, paasns, crb, sas) {
			if err := updateClusterRoleBinding(ctx, r.Client, crb); err != nil {
				return err
			}
		}
	}
	return nil
}

func subjectsFromCrb(crb rbac.ClusterRoleBinding) []string {
	var subjects []string
	for _, subject := range crb.Subjects {
		subjects = append(subjects, fmt.Sprintf("%s/%s/%s", subject.Kind, subject.Namespace, subject.Name))
	}
	return subjects
}

func (r *PaasReconciler) FinalizeExtraClusterRoleBindings(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (err error) {
	ctx, logger := logging.GetLogComponent(ctx, "clusterrolebinding")
	var capRoles []string
	for _, capConfig := range config.GetConfig().Spec.Capabilities {
		capRoles = append(capRoles, capConfig.ExtraPermissions.Roles()...)
		capRoles = append(capRoles, capConfig.DefaultPermissions.Roles()...)
	}
	for _, role := range capRoles {
		roleName := fmt.Sprintf(crbNameFormat, role)
		crb, err := getClusterRoleBinding(ctx, r.Client, role)
		if err != nil {
			return err
		}
		nsRe := fmt.Sprintf("^%s-", paas.Name)
		logger.Info().Msgf("subjects before update: %s", strings.Join(subjectsFromCrb(*crb), ", "))
		changed := updateClusterRoleBindingForRemovedSA(crb,
			*regexp.MustCompile(nsRe), "")
		logger.Info().Msgf("subjects after update: %s", strings.Join(subjectsFromCrb(*crb), ", "))
		if !changed {
			logger.Info().Msg("no changes")
			continue
		}
		logger.Info().Msgf("updating rolebinding %s after cleaning SA's for '%s'", roleName, nsRe)
		if err := updateClusterRoleBinding(ctx, r.Client, crb); err != nil {
			return err
		}
	}
	return nil
}

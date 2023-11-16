package controllers

import (
	"context"
	"fmt"
	"regexp"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ensureClusterRoleBinding ensures ClusterRoleBindings to enable extra permissions for certain capabilities.
func getClusterRoleBinding(
	r client.Client,
	ctx context.Context,
	paas *v1alpha1.Paas,
	name string,
	role string,
) (crb *rbac.ClusterRoleBinding, err error) {
	// See if rolebinding exists and create if it doesn't
	found := &rbac.ClusterRoleBinding{}
	err = r.Get(ctx, types.NamespacedName{Name: name}, found)
	if err != nil && errors.IsNotFound(err) {
		return newClusterRoleBinding(r, paas, name, role), nil
	} else if err != nil {
		// Error that isn't due to the rolebinding not existing
		paas.Status.AddMessage(v1alpha1.PaasStatusError,
			v1alpha1.PaasStatusFind, newClusterRoleBinding(r, paas, name, role), err.Error())
		return nil, err
	} else {
		return found, nil
	}
}

func updateClusterRoleBinding(
	r client.Client,
	ctx context.Context,
	paas *v1alpha1.Paas,
	crb *rbac.ClusterRoleBinding,
) (err error) {
	logger := getLogger(ctx, paas, "ClusterRoleBinding", crb.Name)
	if len(crb.Subjects) == 0 && crb.ResourceVersion != "" {
		logger.Info(fmt.Sprintf("Cleaning empty ClusterRoleBinding %s", crb.Name))
		return r.Delete(ctx, crb)
	} else if len(crb.Subjects) != 0 && crb.ResourceVersion == "" {
		logger.Info(fmt.Sprintf("Creating new ClusterRoleBinding %s", crb.Name))
		return r.Create(ctx, crb)
	} else if len(crb.Subjects) != 0 {
		logger.Info(fmt.Sprintf("Updating existing ClusterRoleBinding %s", crb.Name))
		return r.Update(ctx, crb)
	}
	return nil
}

// backendRoleBinding is a code for Creating RoleBinding
func newClusterRoleBinding(
	r client.Client,
	paas *v1alpha1.Paas,
	name string,
	role string,
) *rbac.ClusterRoleBinding {

	rb := &rbac.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
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

func addSAsToClusterRoleBinding(
	crb *rbac.ClusterRoleBinding,
	namespace string,
	serviceAccounts []string,
) (changed bool) {
	sas := make(map[string]bool)
	// First create a list of all service accounts that where already added for this namespace to this CRB
	for _, subject := range crb.Subjects {
		if subject.Kind != "ServiceAccount" {
			continue
		} else if subject.Namespace != namespace {
			continue
		}
		sas[subject.Name] = true
	}
	for _, sa := range serviceAccounts {
		// Check if the service account is already added
		if _, exists := sas[sa]; exists {
			continue
		}
		// If it was not already in the list, lets add it
		crb.Subjects = append(crb.Subjects, rbac.Subject{
			Kind:      "ServiceAccount",
			Name:      sa,
			Namespace: namespace,
		})
		changed = true
	}
	return changed
}

func updateClusterRoleBindingForRemovedSAs(
	crb *rbac.ClusterRoleBinding,
	nsRe regexp.Regexp,
) (changed bool) {
	var newSubjects []rbac.Subject
	for _, subject := range crb.Subjects {
		if nsRe.MatchString(subject.Namespace) {
			// Subject ns starts with prefix, don't keep.
			changed = true
			continue
		}
		newSubjects = append(newSubjects, subject)
	}
	crb.Subjects = newSubjects
	return changed
}

func (r *PaasNSReconciler) ReconcileExtraClusterRoleBinding(
	ctx context.Context,
	paasns *v1alpha1.PaasNS,
	paas *v1alpha1.Paas,
) (err error) {
	var crb *rbac.ClusterRoleBinding
	var changed bool
	if cap, exists := paas.Spec.Capabilities.AsMap()[paas.Name]; !exists {
		return
	} else if capConfig, exists := getConfig().Capabilities[paas.Name]; !exists {
		return
	} else {

		logger := getLogger(ctx, paas, "ClusterRoleBinding", "reconcile")

		capNamespace := fmt.Sprintf("%s-%s", paas.Name, paas.Name)
		// logger.Info(fmt.Sprintf("capNamespace %s", capNamespace))
		for _, role := range capConfig.ExtraPermissions.Roles {
			crbName := fmt.Sprintf("paas-%s", role)
			// logger.Info(fmt.Sprintf("crbname %s", crbName))
			if crb, err = getClusterRoleBinding(r.Client, ctx, paas, crbName, role); err != nil {
				// logger.Info(fmt.Sprintf("error: %s", err.Error()))
				return err
				// } else {
				// logger.Info(fmt.Sprintf("crb %s read from k8s", crbName))
			}

			if cap.WithExtraPermissions() {
				if changed = addSAsToClusterRoleBinding(crb, capNamespace, capConfig.ExtraPermissions.ServiceAccounts); changed {
					logger.Info(fmt.Sprintf("adding sa's %v for ns %s to crb %s", capConfig.ExtraPermissions.ServiceAccounts, capNamespace, crbName))
				}
			} else {
				if changed = updateClusterRoleBindingForRemovedSAs(crb, *regexp.MustCompile(fmt.Sprintf("^%s$", capNamespace))); changed {
					logger.Info(fmt.Sprintf("deleting sa's %v for ns %s from crb %s", capConfig.ExtraPermissions.ServiceAccounts, capNamespace, crbName))
				}
			}
			if changed {
				// logger.Info(fmt.Sprintf("updating crb %s", crbName))
				if err := updateClusterRoleBinding(r.Client, ctx, paas, crb); err != nil {
					// logger.Info(fmt.Sprintf("updating crb %s failed", crbName))
					return err
				}
			}
		}
	}
	return nil
}

func (r *PaasReconciler) FinalizeExtraClusterRoleBindings(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (err error) {
	logger := getLogger(ctx, paas, "ClusterRoleBinding", "finalize")
	var capRoles []string
	for _, capConfig := range getConfig().Capabilities {
		capRoles = append(capRoles, capConfig.ExtraPermissions.Roles...)
	}
	for _, role := range capRoles {
		roleName := fmt.Sprintf("paas-%s", role)
		crb, err := getClusterRoleBinding(r.Client, ctx, paas, roleName, role)
		if err != nil {
			return err
		}
		nsRe := fmt.Sprintf("^%s-", paas.Name)
		changed := updateClusterRoleBindingForRemovedSAs(crb,
			*regexp.MustCompile(nsRe))
		if !changed {
			continue
		}
		logger.Info(fmt.Sprintf("Updating rolebinding %s after cleaning SA's for '%s'", roleName, nsRe))
		if err := updateClusterRoleBinding(r.Client, ctx, paas, crb); err != nil {
			return err
		}
	}
	return nil
}

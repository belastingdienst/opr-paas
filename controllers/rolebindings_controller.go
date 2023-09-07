package controllers

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ensureRoleBinding ensures RoleBinding presence in given rolebinding.
func (r *PaasReconciler) EnsureAdminRoleBinding(
	ctx context.Context,
	rb *rbac.RoleBinding,
) error {

	// See if rolebinding exists and create if it doesn't
	found := &rbac.RoleBinding{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      rb.Name,
		Namespace: rb.Namespace,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the rolebinding
		err = r.Create(ctx, rb)

		if err != nil {
			// creating the rolebinding failed
			return err
		} else {
			// creating the rolebinding was successful
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the rolebinding not existing
		return err
	}

	return nil
}

// backendRoleBinding is a code for Creating RoleBinding
func (r *PaasReconciler) backendRoleBinding(
	ctx context.Context,
	paas *v1alpha1.Paas,
	name types.NamespacedName,
	groups []string,
) *rbac.RoleBinding {
	logger := getLogger(ctx, paas, "RoleBinding", name.String())
	logger.Info(fmt.Sprintf("Defining %s RoleBinding", name))
	//matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}

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
	return rb
}

func (r *PaasReconciler) BackendEnabledRoleBindings(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (rb []*rbac.RoleBinding) {
	groupKeys := paas.Spec.Groups.AsGroups().Keys()
	for cap_name, cap := range paas.Spec.Capabilities.AsMap() {
		if cap.IsEnabled() {
			name := types.NamespacedName{
				Name:      "paas-admin",
				Namespace: fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, cap_name),
			}
			rb = append(rb, r.backendRoleBinding(ctx, paas, name, groupKeys))
		}
	}
	for _, ns_suffix := range paas.Spec.Namespaces {
		name := types.NamespacedName{
			Name:      "paas-admin",
			Namespace: fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, ns_suffix),
		}
		rb = append(rb, r.backendRoleBinding(ctx, paas, name, groupKeys))
	}
	return rb
}
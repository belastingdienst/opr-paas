/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

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
	paasns *v1alpha1.PaasNS,
	statusMessages *v1alpha1.PaasNsStatus,
	rb *rbac.RoleBinding,
) error {
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
		err = r.Create(ctx, rb)

		if err != nil {
			// creating the rolebinding failed
			statusMessages.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, rb, err.Error())
			return err
		} else {
			// creating the rolebinding was successful
			statusMessages.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, rb, "succeeded")
			return err
		}
	} else if err != nil {
		// Error that isn't due to the rolebinding not existing
		statusMessages.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, rb, err.Error())
		return err
	}
	var changed bool
	if !paasns.AmIOwner(found.OwnerReferences) {
		statusMessages.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "updating owner")
		controllerutil.SetControllerReference(paasns, found, r.GetScheme())
		changed = true
	}
	var ist []string
	for _, subj := range found.Subjects {
		ist = append(ist, subj.Name)
	}
	var sol []string
	for _, subj := range rb.Subjects {
		sol = append(sol, subj.Name)
	}
	if diffRbacSubjects(found.Subjects, rb.Subjects) {
		statusMessages.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found,
			fmt.Sprintf("updating subjects. %s: [%s], %s: [%s]", "ist", strings.Join(ist, ", "), "sol", strings.Join(sol, ", ")))
		found.Subjects = rb.Subjects
		changed = true
	} else {
		statusMessages.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found,
			fmt.Sprintf("updating subjects not needed. %s: [%s], %s: [%s]", "ist", strings.Join(ist, ", "), "sol", strings.Join(sol, ", ")))
	}
	if changed {
		statusMessages.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "updating resource")
		if err = r.Update(ctx, found); err != nil {
			statusMessages.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusUpdate, rb, err.Error())
			return err
		}
	}
	statusMessages.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "succeeded")
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
			Name:     role,
		},
	}
	logger.Info("Setting Owner")
	controllerutil.SetControllerReference(paas, rb, r.GetScheme())
	return rb
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

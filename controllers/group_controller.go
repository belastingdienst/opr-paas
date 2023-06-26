package controllers

import (
	"context"
	"fmt"

	mydomainv1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"

	userv1 "github.com/openshift/api/user/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureGroup ensures Group presence
func (r *PaasReconciler) ensureGroup(
	group *userv1.Group,
) error {

	// See if group already exists and create if it doesn't
	found := &userv1.Group{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Name: group.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the group
		err = r.Create(context.TODO(), group)

		if err != nil {
			// creating the group failed
			return err
		} else {
			// creating the group was successful
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the group not existing
		return err
	}

	return nil
}

// backendGroup is a code for Creating Group
func (r *PaasReconciler) backendGroup(paas *mydomainv1alpha1.Paas, groupName string, users []string) *userv1.Group {
	//matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}
	group := &userv1.Group{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Group",
			APIVersion: "user.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   groupName,
			Labels: paas.Labels,
		},
		Users: users,
	}

	//If we would have multiple PaaS projects defining this group, and all are cleaned,
	//the garbage collector would also clean this group...
	controllerutil.SetControllerReference(paas, group, r.Scheme)
	return group
}

func (r *PaasReconciler) backendGroups(paas *mydomainv1alpha1.Paas) (groups []*userv1.Group) {
	for key, group := range paas.Spec.Groups {
		groups = append(groups, r.backendGroup(paas, paas.Spec.Groups.NameFromQuery(key), group.Users))
	}
	return groups
}

// cleanGroup is a code for Creating Group
func (r *PaasReconciler) finalizeGroup(paas *mydomainv1alpha1.Paas, groupName string) (cleaned bool, err error) {
	obj := &userv1.Group{}
	if err := r.Get(context.TODO(), types.NamespacedName{
		Name: groupName,
	}, obj); err != nil && errors.IsNotFound(err) {
		fmt.Printf("%s does not exist", groupName)
		return false, nil
	} else if err != nil {
		fmt.Printf("%s not deleted, error", groupName)
		return false, err
	} else if !paas.AmIOwner(obj.OwnerReferences) {
		return false, nil
	} else {
		obj.OwnerReferences = paas.WithoutMe(obj.OwnerReferences)
		if len(obj.OwnerReferences) == 0 {
			fmt.Printf("%s trying to delete", groupName)
			return true, r.Delete(context.TODO(), obj)
		}
	}
	return false, nil
}

func (r *PaasReconciler) finalizeGroups(paas *mydomainv1alpha1.Paas) (cleaned []string, err error) {
	for key, group := range paas.Spec.Groups {
		if isCleaned, err := r.finalizeGroup(paas, paas.Spec.Groups.NameFromQuery(key)); err != nil {
			return cleaned, err
		} else if isCleaned && group.Query != "" {
			cleaned = append(cleaned, group.Query)
		}
	}
	return cleaned, nil
}

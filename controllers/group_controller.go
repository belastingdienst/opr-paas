package controllers

import (
	"context"

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
	for group, users := range paas.Spec.Groups {
		groups = append(groups, r.backendGroup(paas, group, users))
	}
	return groups
}

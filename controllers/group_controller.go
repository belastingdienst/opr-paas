package controllers

import (
	"context"
	"fmt"
	"os"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	userv1 "github.com/openshift/api/user/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureGroup ensures Group presence
func (r *PaasReconciler) EnsureGroup(
	ctx context.Context,
	group *userv1.Group,
) error {

	// See if group already exists and create if it doesn't
	found := &userv1.Group{}
	err := r.Get(ctx, types.NamespacedName{
		Name: group.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the group
		err = r.Create(ctx, group)

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
func (r *PaasReconciler) backendGroup(
	ctx context.Context,
	paas *v1alpha1.Paas,
	name string,
	group v1alpha1.PaasGroup,
) *userv1.Group {
	logger := getLogger(ctx, paas, "Group", name)
	logger.Info("Defining group")

	labels := make(map[string]string)
	for key, value := range paas.Labels {
		labels[key] = value
	}
	labels["openshift.io/ldap.host"] = os.Getenv("LDAP_HOST")

	annotations := make(map[string]string)
	annotations["openshift.io/ldap.uid"] = group.Query
	annotations["openshift.io/ldap.url"] = fmt.Sprintf("%s:%s",
		os.Getenv("LDAP_HOST"),
		os.Getenv("LDAP_PORT"),
	)

	//matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}
	g := &userv1.Group{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Group",
			APIVersion: "user.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
		Users: group.Users,
	}

	//If we would have multiple PaaS projects defining this group, and all are cleaned,
	//the garbage collector would also clean this group...
	controllerutil.SetControllerReference(paas, g, r.Scheme)
	return g
}

func (r *PaasReconciler) BackendGroups(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (groups []*userv1.Group) {
	logger := getLogger(ctx, paas, "Group", "")
	for key, group := range paas.Spec.Groups {
		groupName := paas.Spec.Groups.Key2Name(key)
		logger.Info("Groupname is " + groupName)
		groups = append(groups, r.backendGroup(ctx,
			paas,
			groupName,
			group))
	}
	return groups
}

// cleanGroup is a code for Creating Group
func (r *PaasReconciler) finalizeGroup(
	ctx context.Context,
	paas *v1alpha1.Paas,
	groupName string,
) (cleaned bool, err error) {
	logger := getLogger(ctx, paas, "Group", groupName)
	obj := &userv1.Group{}
	if err := r.Get(ctx, types.NamespacedName{
		Name: groupName,
	}, obj); err != nil && errors.IsNotFound(err) {
		logger.Info("Group does not exist")
		return false, nil
	} else if err != nil {
		logger.Info("Group not deleted. error: " + err.Error())
		return false, err
	} else if !paas.AmIOwner(obj.OwnerReferences) {
		logger.Info("Paas is not an owner")
		return false, nil
	} else {
		logger.Info(groupName + "Removing PaaS finalizer")
		obj.OwnerReferences = paas.WithoutMe(obj.OwnerReferences)
		if len(obj.OwnerReferences) == 0 {
			logger.Info(groupName + "Deleting")
			return true, r.Delete(ctx, obj)
		}
	}
	return false, nil
}

func (r *PaasReconciler) FinalizeGroups(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (cleaned []string, err error) {
	for key, group := range paas.Spec.Groups {
		if isCleaned, err := r.finalizeGroup(ctx, paas, paas.Spec.Groups.Key2Name(key)); err != nil {
			return cleaned, err
		} else if isCleaned && group.Query != "" {
			cleaned = append(cleaned, group.Query)
		}
	}
	return cleaned, nil
}

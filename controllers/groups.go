/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	"github.com/go-logr/logr"
	userv1 "github.com/openshift/api/user/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func uniqueUsers(found userv1.OptionalNames, expected userv1.OptionalNames) (unique userv1.OptionalNames) {

	// All of this is to make the list of users a unique
	// combined list of users that where and now should be added
	users := make(map[string]bool)
	for _, user := range found {
		users[user] = true
	}
	for _, user := range expected {
		users[user] = true
	}
	for user := range users {
		unique = append(unique, user)
	}
	return
}

func mergeStringMap(first map[string]string, second map[string]string) map[string]string {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}

	for key, value := range second {
		first[key] = value
	}
	return first
}

// ensureGroup ensures Group presence
func (r *PaasReconciler) EnsureGroup(
	ctx context.Context,
	paas *v1alpha1.Paas,
	group *userv1.Group,
) error {

	logger := getLogger(ctx, paas, "Group", group.Name)

	// See if group already exists and create if it doesn't
	found := &userv1.Group{}
	err := r.Get(ctx, types.NamespacedName{
		Name: group.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating the group")
		// Create the group
		if err = r.Create(ctx, group); err != nil {
			// creating the group failed
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, group, err.Error())
			return err
		} else {
			// creating the group was successful
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, group, "succeeded")
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the group not existing
		logger.Error(err, "Could not retrieve info on the group")
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, group, err.Error())
		return err
	}
	logger.Info("Updating the group")
	// All of this is to make the list of users a unique
	// combined list of users that where and now should be added
	found.Users = uniqueUsers(found.Users, group.Users)
	found.Annotations = mergeStringMap(found.Annotations, group.Annotations)
	if !paas.AmIOwner(found.OwnerReferences) {
		logger.Info("Setting owner reference")
		controllerutil.SetOwnerReference(paas, found, r.Scheme)
	} else {
		logger.Info("Already owner")
	}
	if err = r.Update(ctx, found); err != nil {
		// Updating the group failed
		logger.Error(err, "Updating the group failed")
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusUpdate, group, err.Error())
		return err
	} else {
		logger.Info("Group updated")
		// Updating the group was successful
		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, group, "succeeded")
		return nil
	}
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

	g := &userv1.Group{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Group",
			APIVersion: "user.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: paas.ClonedLabels(),
			Annotations: map[string]string{
				"openshift.io/ldap.uid": group.Query,
				"openshift.io/ldap.url": fmt.Sprintf("%s:%d",
					getConfig().LDAP.Host,
					getConfig().LDAP.Port,
				),
			},
		},
		Users: group.Users,
	}
	g.ObjectMeta.Labels["openshift.io/ldap.host"] = getConfig().LDAP.Host

	//If we would have multiple PaaS projects defining this group, and all are cleaned,
	//the garbage collector would also clean this group...
	controllerutil.SetOwnerReference(paas, g, r.Scheme)
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
		logger.Info("Removing PaaS finalizer " + groupName)
		obj.OwnerReferences = paas.WithoutMe(obj.OwnerReferences)
		if len(obj.OwnerReferences) == 0 {
			logger.Info("Deleting " + groupName)
			return true, r.Delete(ctx, obj)
		} else {
			logger.Info("Not last reference, skipping deletion for " + groupName)
			return false, r.Update(ctx, obj)
		}
	}
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

func (r *PaasReconciler) ReconcileGroups(
	ctx context.Context,
	paas *v1alpha1.Paas,
	logger logr.Logger,
) error {

	logger.Info("Creating groups for PAAS object ")
	for _, group := range r.BackendGroups(ctx, paas) {
		if err := r.EnsureGroup(ctx, paas, group); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating group %s", group.ObjectMeta.Name))
			return err
		}
	}
	return nil
}

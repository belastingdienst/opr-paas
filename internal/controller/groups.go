/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	userv1 "github.com/openshift/api/user/v1"
	"github.com/rs/zerolog/log"
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
	logger := log.Ctx(ctx)

	// See if group already exists and create if it doesn't
	found := &userv1.Group{}
	err := r.Get(ctx, types.NamespacedName{
		Name: group.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info().Msg("creating the group")
		// Create the group
		if err = r.Create(ctx, group); err != nil {
			// creating the group failed
			return err
		}
		return nil
	} else if err != nil {
		// Error that isn't due to the group not existing
		logger.Err(err).Msg("could not retrieve info on the group")
		return err
	}
	logger.Info().Msg("updating the group")
	// All of this is to make the list of users a unique
	// combined list of users that where and now should be added
	found.Users = uniqueUsers(found.Users, group.Users)
	found.Annotations = mergeStringMap(found.Annotations, group.Annotations)
	if !paas.AmIOwner(found.OwnerReferences) {
		logger.Info().Msg("setting owner reference")
		if err := controllerutil.SetOwnerReference(paas, found, r.Scheme); err != nil {
			logger.Err(err).Msg("error while setting owner reference")
		}
	} else {
		logger.Info().Msg("already owner")
	}
	if err = r.Update(ctx, found); err != nil {
		// Updating the group failed
		logger.Err(err).Msg("updating the group failed")
		return err
	} else {
		logger.Info().Msg("group updated")
		// Updating the group was successful
		return nil
	}
}

// backendGroup is a code for Creating Group
func (r *PaasReconciler) backendGroup(
	ctx context.Context,
	paas *v1alpha1.Paas,
	name string,
	group v1alpha1.PaasGroup,
) (*userv1.Group, error) {
	logger := log.Ctx(ctx)
	logger.Info().Msg("defining group")

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
					GetConfig().LDAP.Host,
					GetConfig().LDAP.Port,
				),
			},
		},
		Users: group.Users,
	}
	g.ObjectMeta.Labels["openshift.io/ldap.host"] = GetConfig().LDAP.Host

	// If we had multiple Paas projects defining this group, and all are cleaned,
	// the garbage collector would also clean this group...
	if err := controllerutil.SetOwnerReference(paas, g, r.Scheme); err != nil {
		return g, err
	}
	return g, nil
}

func (r *PaasReconciler) BackendGroups(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (groups []*userv1.Group) {
	logger := log.Ctx(ctx)
	for key, group := range paas.Spec.Groups {
		groupName := paas.Spec.Groups.Key2Name(key)
		logger.Info().Msg("groupname is " + groupName)
		beGroup, _ := r.backendGroup(ctx, paas, groupName, group)
		groups = append(groups, beGroup)
	}
	return groups
}

func (r *PaasReconciler) FinalizeGroups(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	ctx = setLogComponent(ctx, "group")
	existingGroups, err := r.getExistingGroups(ctx, paas)
	if err != nil {
		return err
	}
	removedLdapGroups, err := r.deleteObsoleteGroups(ctx, paas, []*userv1.Group{}, existingGroups)
	if err != nil {
		return err
	}
	if len(removedLdapGroups) != 0 {
		err = r.FinalizeLdapGroups(ctx, removedLdapGroups)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *PaasReconciler) ReconcileGroups(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	ctx = setLogComponent(ctx, "group")
	logger := log.Ctx(ctx)
	logger.Info().Msg("reconciling groups for Paas")
	desiredGroups := r.BackendGroups(ctx, paas)
	existingGroups, err := r.getExistingGroups(ctx, paas)
	if err != nil {
		return err
	}
	removedLdapGroups, err := r.deleteObsoleteGroups(ctx, paas, desiredGroups, existingGroups)
	if err != nil {
		return err
	}
	if len(removedLdapGroups) != 0 {
		err = r.FinalizeLdapGroups(ctx, removedLdapGroups)
		if err != nil {
			return err
		}
	}
	for _, group := range desiredGroups {
		if err := r.EnsureGroup(ctx, paas, group); err != nil {
			logger.Err(err).Msgf("failure while reconciling group %s", group.ObjectMeta.Name)
			return err
		}
	}
	return nil
}

// deleteObsoleteGroups delete groups which are no longer desired from a Paas desired state. As multiple Paas'es can reference one
// group, a check is executed whether the group can really be removed. If a Group is marked as an ldap group, the ldap query is added
// to a list of to be removedLdapGroups.
func (r *PaasReconciler) deleteObsoleteGroups(ctx context.Context, paas *v1alpha1.Paas, desiredGroups []*userv1.Group, existingGroups []*userv1.Group) (removedLdapGroups []string, err error) {
	logger := log.Ctx(ctx)
	logger.Info().Msg("deleting obsolete groups")
	// Delete groups that are no longer needed
	for _, existingGroup := range existingGroups {
		if !isGroupInGroups(existingGroup, desiredGroups) {
			existingGroup.OwnerReferences = paas.WithoutMe(existingGroup.OwnerReferences)
			if len(existingGroup.OwnerReferences) == 0 {
				logger.Info().Msgf("deleting %s", existingGroup.Name)
				if err = r.Delete(ctx, existingGroup); err != nil {
					return removedLdapGroups, err
				}
				if existingGroup.Annotations["openshift.io/ldap.uid"] != "" {
					removedLdapGroups = append(removedLdapGroups, existingGroup.Annotations["openshift.io/ldap.uid"])
				}
				continue
			}
			logger.Info().Msgf("not last owner of group %s", existingGroup.Name)
			// FIXME no updates to users is executed
			err = r.Update(ctx, existingGroup)
			if err != nil {
				return removedLdapGroups, err
			}
		}
	}
	return removedLdapGroups, nil
}

// isGroupInGroups determines whether a list of groups contains a specified group, based on it's name
func isGroupInGroups(group *userv1.Group, groups []*userv1.Group) bool {
	for _, desiredGroup := range groups {
		if group.Name == desiredGroup.Name {
			return true
		}
	}
	return false
}

// TODO (portly-halicore-76) use label selector as this is quite expensive
// getExistingGroups returns all groups owned by the specified Paas
func (r *PaasReconciler) getExistingGroups(ctx context.Context, paas *v1alpha1.Paas) (existingGroups []*userv1.Group, err error) {
	logger := log.Ctx(ctx)
	var groups userv1.GroupList
	err = r.List(ctx, &groups)
	if err != nil {
		return existingGroups, err
	}
	for _, group := range groups.Items {
		if paas.AmIOwner(group.OwnerReferences) {
			logger.Debug().Msgf("existing group %s owned by Paas %s", group.ObjectMeta.Name, paas.Name)
			existingGroups = append(existingGroups, &group)
		}
		logger.Debug().Msgf("existing group %s not owned by Paas %s", group.ObjectMeta.Name, paas.Name)
	}
	logger.Debug().Msgf("found %d existing groups owned by Paas %s", len(existingGroups), paas.Name)
	return existingGroups, nil
}

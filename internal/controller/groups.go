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
	"sigs.k8s.io/controller-runtime/pkg/client"

	userv1 "github.com/openshift/api/user/v1"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EnsureGroup ensures Group presence
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
		logger.Err(err).Msg("could not retrieve the group")
		return err
	}
	if !paas.AmIOwner(found.OwnerReferences) {
		logger.Info().Msg("setting owner reference")
		if err := controllerutil.SetOwnerReference(paas, found, r.Scheme); err != nil {
			logger.Err(err).Msg("error while setting owner reference")
			return err
		}
		return r.Update(ctx, found)
	}
	return nil
}

// backendGroup returns the desired group, based in the paasGroupKey and the group defined in that key.
// if the paasGroup contains both users and a query, which is mutually exclusive, the query takes precedence.
// groups with users, are made paas specific by prefixing them with the paas.Name
// groups with a query can ben shared between multiple Paas'es referencing the same group.
func (r *PaasReconciler) backendGroup(
	ctx context.Context,
	paas *v1alpha1.Paas,
	paasGroupKey string,
	group v1alpha1.PaasGroup,
) (*userv1.Group, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("defining group")
	g := &userv1.Group{}
	groupName := paas.GroupKey2GroupName(paasGroupKey)
	if len(group.Query) > 0 {
		g.ObjectMeta = metav1.ObjectMeta{
			Name:   groupName,
			Labels: paas.ClonedLabels(),
			Annotations: map[string]string{
				"openshift.io/ldap.uid": group.Query,
				"openshift.io/ldap.url": fmt.Sprintf("%s:%d",
					GetConfig().LDAP.Host,
					GetConfig().LDAP.Port,
				),
			},
		}
		g.ObjectMeta.Labels["openshift.io/ldap.host"] = GetConfig().LDAP.Host
		g.ObjectMeta.Labels["app.kubernetes.io/managed-by"] = "paas"
	} else {
		g.ObjectMeta = metav1.ObjectMeta{
			Name:   groupName,
			Labels: paas.ClonedLabels(),
		}
		g.ObjectMeta.Labels["app.kubernetes.io/managed-by"] = "paas"
		g.Users = group.Users
	}

	if err := controllerutil.SetOwnerReference(paas, g, r.Scheme); err != nil {
		return nil, err
	}
	return g, nil
}

func (r *PaasReconciler) backendGroups(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (groups []*userv1.Group, err error) {
	for key, group := range paas.Spec.Groups {
		beGroup, err := r.backendGroup(ctx, paas, key, group)
		if err != nil {
			return nil, err
		}
		groups = append(groups, beGroup)
	}
	return groups, nil
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
	desiredGroups, err := r.backendGroups(ctx, paas)
	if err != nil {
		return err
	}
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

// deleteObsoleteGroups delete groups which are no longer desired from a Paas desired state.
// If a Group is marked as an LDAP group, and there is no Paas referencing it, the LDAP query is added to a list of to be removedLdapGroups.
func (r *PaasReconciler) deleteObsoleteGroups(ctx context.Context, paas *v1alpha1.Paas, desiredGroups []*userv1.Group, existingGroups []*userv1.Group) (removedLdapGroups []string, err error) {
	logger := log.Ctx(ctx)
	logger.Info().Msg("deleting obsolete groups")
	for _, existingGroup := range existingGroups {
		if !isGroupInGroups(existingGroup, desiredGroups) {
			if existingGroup.Annotations["openshift.io/ldap.uid"] != "" {
				existingGroup.OwnerReferences = paas.WithoutMe(existingGroup.OwnerReferences)
				if len(existingGroup.OwnerReferences) == 0 {
					logger.Info().Msgf("deleting %s", existingGroup.Name)
					if err = r.Delete(ctx, existingGroup); err != nil {
						return removedLdapGroups, err
					}
					removedLdapGroups = append(removedLdapGroups, existingGroup.Annotations["openshift.io/ldap.uid"])
					continue
				}
				logger.Info().Msgf("not last owner of group %s", existingGroup.Name)
				err = r.Update(ctx, existingGroup)
				if err != nil {
					return removedLdapGroups, err
				}
			}
			if err = r.Delete(ctx, existingGroup); err != nil {
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

// getExistingGroups returns all groups owned by the specified Paas
func (r *PaasReconciler) getExistingGroups(ctx context.Context, paas *v1alpha1.Paas) (existingGroups []*userv1.Group, err error) {
	logger := log.Ctx(ctx)
	var groups userv1.GroupList
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{"app.kubernetes.io/managed-by": "paas"}),
	}
	err = r.List(ctx, &groups, listOpts...)
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

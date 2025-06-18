/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"maps"
	"reflect"

	"github.com/belastingdienst/opr-paas/v2/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v2/internal/config"
	"github.com/belastingdienst/opr-paas/v2/internal/logging"
	"github.com/belastingdienst/opr-paas/v2/internal/templating"
	"sigs.k8s.io/controller-runtime/pkg/client"

	userv1 "github.com/openshift/api/user/v1"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// ManagedByLabelKey is the key of the label that specifies the tool being used to manage the operation of this
	// application. For more info, see
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	ManagedByLabelKey = "cpet.belastingdienst.nl/managed-by-paas"
)

func (r *PaasReconciler) ensureGroup(
	ctx context.Context,
	paas *v1alpha2.Paas,
	group *userv1.Group,
) error {
	var (
		changed   bool
		groupName = group.GetName()
	)
	logger := log.Ctx(ctx)
	// See if group already exists and create if it doesn't
	found := &userv1.Group{}
	err := r.Get(ctx, types.NamespacedName{
		Name: group.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info().Msg("creating group " + groupName)
		// Create the group
		if err = r.Create(ctx, group); err != nil {
			// creating the group failed
			return err
		}
		return nil
	} else if err != nil {
		// Error that isn't due to the group not existing
		logger.Err(err).Msg("could not retrieve group " + groupName)
		return err
	}
	if !paas.AmIOwner(found.OwnerReferences) {
		logger.Info().Msg("setting owner reference on group " + groupName)
		if err := controllerutil.SetOwnerReference(paas, found, r.Scheme); err != nil {
			logger.Err(err).Msg("error while setting owner reference on group " + groupName)
			return err
		}
		changed = true
	}
	if reflect.DeepEqual(group.Users, found.Users) {
		logger.Debug().Msg("users for group " + groupName + " are as expected")
	} else {
		found.Users = group.Users
		changed = true
	}
	if !reflect.DeepEqual(found.Labels, group.Labels) {
		logger.Debug().Msg("group " + groupName + " labels changed")
		found.Labels = group.Labels
		changed = true
	}
	if changed {
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
	paas *v1alpha2.Paas,
	paasGroupKey string,
	group v1alpha2.PaasGroup,
) (*userv1.Group, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("defining group")
	// We don't manage groups with a query
	if len(group.Query) != 0 {
		return nil, nil
	}

	labels := map[string]string{}
	myConfig := config.GetConfig()
	labelTemplater := templating.NewTemplater(*paas, myConfig)
	for name, tpl := range myConfig.Spec.ResourceLabels.GroupLabels {
		result, err := labelTemplater.TemplateToMap(name, tpl)
		if err != nil {
			return nil, err
		}
		maps.Copy(labels, result)
	}

	g := &userv1.Group{}
	groupName := paas.GroupKey2GroupName(paasGroupKey)
	g.ObjectMeta = metav1.ObjectMeta{
		Name:   groupName,
		Labels: labels,
	}
	g.Users = group.Users
	g.Labels[ManagedByLabelKey] = paas.Name

	if err := controllerutil.SetOwnerReference(paas, g, r.Scheme); err != nil {
		return nil, err
	}
	return g, nil
}

func (r *PaasReconciler) backendGroups(
	ctx context.Context,
	paas *v1alpha2.Paas,
) (groups []*userv1.Group, err error) {
	for key, group := range paas.Spec.Groups {
		beGroup, err := r.backendGroup(ctx, paas, key, group)
		if err != nil {
			return nil, err
		}
		if beGroup == nil {
			continue // Skip Query groups
		}
		groups = append(groups, beGroup)
	}
	return groups, nil
}

func (r *PaasReconciler) finalizeGroups(
	ctx context.Context,
	paas *v1alpha2.Paas,
) error {
	ctx, _ = logging.GetLogComponent(ctx, "group")
	existingGroups, err := r.getExistingGroups(ctx, paas)
	if err != nil {
		return err
	}
	err = r.deleteObsoleteGroups(ctx, paas, []*userv1.Group{}, existingGroups)
	if err != nil {
		return err
	}
	return nil
}

func (r *PaasReconciler) reconcileGroups(
	ctx context.Context,
	paas *v1alpha2.Paas,
) error {
	ctx, logger := logging.GetLogComponent(ctx, "group")
	logger.Info().Msg("reconciling groups for Paas")
	desiredGroups, err := r.backendGroups(ctx, paas)
	if err != nil {
		return err
	}
	existingGroups, err := r.getExistingGroups(ctx, paas)
	if err != nil {
		return err
	}
	err = r.deleteObsoleteGroups(ctx, paas, desiredGroups, existingGroups)
	if err != nil {
		return err
	}
	for _, group := range desiredGroups {
		if err := r.ensureGroup(ctx, paas, group); err != nil {
			logger.Err(err).Msgf("failure while reconciling group %s", group.Name)
			return err
		}
	}
	return nil
}

// deleteObsoleteGroups delete groups which are no longer desired from a Paas desired state.
// If a Group is marked as an LDAP group, and there is no Paas referencing it,
// the LDAP query is added to a list of to be removedLdapGroups.
func (r *PaasReconciler) deleteObsoleteGroups(
	ctx context.Context,
	paas *v1alpha2.Paas,
	desiredGroups []*userv1.Group,
	existingGroups []*userv1.Group,
) error {
	logger := log.Ctx(ctx)
	logger.Info().Msg("deleting obsolete groups")
	for _, existingGroup := range existingGroups {
		if !isGroupInGroups(existingGroup, desiredGroups) {
			if err := r.Delete(ctx, existingGroup); err != nil {
				return err
			}
		}
	}
	return nil
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
func (r *PaasReconciler) getExistingGroups(
	ctx context.Context,
	paas *v1alpha2.Paas,
) (existingGroups []*userv1.Group, err error) {
	logger := log.Ctx(ctx)
	var groups userv1.GroupList
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{ManagedByLabelKey: paas.Name}),
	}
	err = r.List(ctx, &groups, listOpts...)
	if err != nil {
		return existingGroups, err
	}
	for _, group := range groups.Items {
		existingGroups = append(existingGroups, &group)
	}
	logger.Debug().Msgf("found %d existing groups owned by Paas %s", len(existingGroups), paas.Name)
	return existingGroups, nil
}

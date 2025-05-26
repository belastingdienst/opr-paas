/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/groups"
	"github.com/belastingdienst/opr-paas/internal/logging"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *PaasReconciler) ensureLdapGroupsConfigMap(
	ctx context.Context,
	groupSynListKey string,
) error {
	// Create the ConfigMap
	wlConfigMap := config.GetConfig().Spec.GroupSyncList
	return r.Create(ctx, &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      wlConfigMap.Name,
			Namespace: wlConfigMap.Namespace,
		},
		Data: map[string]string{
			config.GetConfig().Spec.GroupSyncListKey: groupSynListKey,
		},
	})
}

// ensureLdapGroups ensures Group presence
func (r *PaasReconciler) ensureLdapGroups(
	ctx context.Context,
	paas *v1alpha2.Paas,
) error {
	ctx, logger := logging.GetLogComponent(ctx, "ldapgroup")
	logger.Info().Msg("creating ldap groups for PAAS object ")
	// See if group already exists and create if it doesn't
	namespacedName := config.GetConfig().Spec.GroupSyncList
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Configmap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
	}
	err := r.Get(ctx, types.NamespacedName{Namespace: namespacedName.Namespace, Name: namespacedName.Name}, cm)
	gs := paas.Spec.Groups.AsGroups()
	if err != nil && errors.IsNotFound(err) {
		logger.Info().Msg("creating groupsynclist configmap")
		// Create the ConfigMap
		return r.ensureLdapGroupsConfigMap(ctx, gs.AsString())
	} else if err != nil {
		logger.Err(err).Msg("could not retrieve groupsynclist configmap")
		return err
	} else if groupsynclist, exists := cm.Data[config.GetConfig().Spec.GroupSyncListKey]; !exists {
		logger.Info().Msg("adding groupsynclist.txt to groupsynclist configmap")
		cm.Data[config.GetConfig().Spec.GroupSyncListKey] = gs.AsString()
	} else {
		logger.Info().Msgf("reading group queries from groupsynclist %v", cm)
		groupsynclistGroups := groups.NewGroups()
		groupsynclistGroups.AddFromString(groupsynclist)
		logger.Info().Msgf("adding extra groups to groupsynclist: %v", gs)
		if changed := groupsynclistGroups.Add(&gs); !changed {
			logger.Info().Msg("no new info in groupsynclist")
			return nil
		}
		logger.Info().Msg("adding to groupsynclist configmap")
		cm.Data[config.GetConfig().Spec.GroupSyncListKey] = groupsynclistGroups.AsString()
	}
	logger.Info().Msgf("updating groupsynclist configmap: %v", cm)
	return r.Update(ctx, cm)
}

// ensureLdapGroup ensures Group presence
func (r *PaasReconciler) finalizeLdapGroups(
	ctx context.Context,
	cleanedLdapQueries []string,
) error {
	var groupSyncList string
	var exists bool
	ctx, logger := logging.GetLogComponent(ctx, "ldapgroup")
	cm := &corev1.ConfigMap{}
	wlConfigMap := config.GetConfig().Spec.GroupSyncList
	err := r.Get(ctx, types.NamespacedName{Name: wlConfigMap.Name, Namespace: wlConfigMap.Namespace}, cm)
	if err != nil && errors.IsNotFound(err) {
		logger.Info().Msg("groupsynclist configmap does not exist")
		// ConfigMap does not exist, so nothing to clean
		return nil
	} else if err != nil {
		logger.Err(err).Msg("error retrieving groupsynclist configmap")
		// Error that isn't due to the group not existing
		return err
	} else if groupSyncList, exists = cm.Data[config.GetConfig().Spec.GroupSyncListKey]; !exists {
		// No groupsynclist.txt exists in the configmap, so nothing to clean
		logger.Info().Msgf("%s does not exists in groupsynclist configmap", config.GetConfig().Spec.GroupSyncListKey)
		return nil
	}
	var isChanged bool
	gs := groups.NewGroups()
	gs.AddFromString(groupSyncList)
	for _, query := range cleanedLdapQueries {
		g := groups.NewGroup(query)
		if g.Key == "" {
			logger.Info().Str("query", query).Msg("could not get key")
		} else if gs.DeleteByKey(g.Key) {
			logger.Info().Msgf("ldapGroup %s removed", g.Key)
			isChanged = true
		}
	}
	if !isChanged {
		return nil
	}
	cm.Data[config.GetConfig().Spec.GroupSyncListKey] = gs.AsString()
	return r.Update(ctx, cm)
}

/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/groups"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	whitelistKeyName = "whitelist.txt"
)

func (r *PaasReconciler) ensureLdapGroupsConfigMap(
	ctx context.Context,
	groups string,
) error {
	// Create the ConfigMap
	wlConfigMap := GetConfig().Whitelist
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
			whitelistKeyName: groups,
		},
	})
}

// ensureLdapGroup ensures Group presence
func (r *PaasReconciler) EnsureLdapGroups(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	ctx = setLogComponent(ctx, "ldapgroup")
	logger := log.Ctx(ctx)
	logger.Info().Msg("creating ldap groups for PAAS object ")
	// See if group already exists and create if it doesn't
	namespacedName := GetConfig().Whitelist
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
		logger.Info().Msg("creating whitelist configmap")
		// Create the ConfigMap
		return r.ensureLdapGroupsConfigMap(ctx, gs.AsString())
	} else if err != nil {
		logger.Err(err).Msg("could not retrieve whitelist configmap")
		return err
	} else if whitelist, exists := cm.Data[whitelistKeyName]; !exists {
		logger.Info().Msg("adding whitelist.txt to whitelist configmap")
		cm.Data[whitelistKeyName] = gs.AsString()
	} else {
		logger.Info().Msgf("reading group queries from whitelist %v", cm)
		whitelistGroups := groups.NewGroups()
		whitelistGroups.AddFromString(whitelist)
		logger.Info().Msgf("adding extra groups to whitelist: %v", gs)
		if changed := whitelistGroups.Add(&gs); !changed {
			logger.Info().Msg("no new info in whitelist")
			return nil
		}
		logger.Info().Msg("adding to whitelist configmap")
		cm.Data[whitelistKeyName] = whitelistGroups.AsString()
	}
	logger.Info().Msgf("updating whitelist configmap: %v", cm)
	return r.Update(ctx, cm)
}

// ensureLdapGroup ensures Group presence
func (r *PaasReconciler) FinalizeLdapGroups(
	ctx context.Context,
	cleanedLdapQueries []string,
) error {
	ctx = setLogComponent(ctx, "ldapgroup")
	logger := log.Ctx(ctx)
	// See if group already exists and create if it doesn't
	cm := &corev1.ConfigMap{}
	wlConfigMap := GetConfig().Whitelist
	err := r.Get(ctx, types.NamespacedName{Name: wlConfigMap.Name, Namespace: wlConfigMap.Namespace}, cm)
	if err != nil && errors.IsNotFound(err) {
		logger.Info().Msg("whitelist configmap does not exist")
		// ConfigMap does not exist, so nothing to clean
		return nil
	} else if err != nil {
		logger.Err(err).Msg("error retrieving whitelist configmap")
		// Error that isn't due to the group not existing
		return err
	} else if whitelist, exists := cm.Data[whitelistKeyName]; !exists {
		// No whitelist.txt exists in the configmap, so nothing to clean
		logger.Info().Msgf("%s does not exists in whitelist configmap", whitelistKeyName)
		return nil
	} else {
		var isChanged bool
		gs := groups.NewGroups()
		gs.AddFromString(whitelist)
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
		cm.Data[whitelistKeyName] = gs.AsString()
	}
	return r.Update(ctx, cm)
}

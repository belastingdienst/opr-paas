/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

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
	wlConfigMap := GetConfig().Spec.Whitelist
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
	namespacedName := GetConfig().Spec.Whitelist
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
	err := r.Get(ctx, namespacedName, cm)
	gs := paas.Spec.Groups.AsGroups()
	if err != nil && errors.IsNotFound(err) {
		logger.Info().Msg("creating whitelist configmap")
		// Create the ConfigMap
		if err = r.ensureLdapGroupsConfigMap(ctx, gs.AsString()); err != nil {
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, cm, err.Error())
		} else {
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, cm, "succeeded")
		}
		return err
	} else if err != nil {
		logger.Err(err).Msg("could not retrieve whitelist configmap")
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, cm, err.Error())
		// Error that isn't due to the group not existing
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
			// fmt.Printf("configured: %d, combined: %d", l1, l2)
			logger.Info().Msg("no new info in whitelist")
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, cm, "no changes")
			return nil
		}
		logger.Info().Msg("adding to whitelist configmap")
		cm.Data[whitelistKeyName] = whitelistGroups.AsString()
	}
	logger.Info().Msgf("updating whitelist configmap: %v", cm)
	if err = r.Update(ctx, cm); err != nil {
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusUpdate, cm, err.Error())
	} else {
		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, cm, "succeeded")
	}
	return err
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
	wlConfigMap := GetConfig().Spec.Whitelist
	err := r.Get(ctx, wlConfigMap, cm)
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
		// fmt.Printf("configured: %d, combined: %d", l1, l2)
		if !isChanged {
			return nil
		}
		cm.Data[whitelistKeyName] = gs.AsString()
	}
	return r.Update(ctx, cm)
}

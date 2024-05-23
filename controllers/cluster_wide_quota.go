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
	paas_quota "github.com/belastingdienst/opr-paas/internal/quota"
	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *PaasReconciler) FetchAllPaasCapabilityResources(
	ctx context.Context,
	quota *quotav1.ClusterResourceQuota,
	defaults map[string]string,
) (resources paas_quota.QuotaLists, err error) {
	paas := &v1alpha1.Paas{}
	resources = paas_quota.NewQuotaLists()
	for _, reference := range quota.OwnerReferences {
		paasNamespacedName := types.NamespacedName{Name: reference.Name}
		if reference.Kind != "Paas" || reference.APIVersion != v1alpha1.GroupVersion.String() {
			err = fmt.Errorf("quota references a missing paas")
			return
		} else if err = r.Get(ctx, paasNamespacedName, paas); err != nil {
			return
		} else if paasCap, exists := paas.Spec.Capabilities.AsMap()[quota.Name]; !exists {
			resources.Append(paas_quota.NewQuota(defaults))
		} else {
			resources.Append(paasCap.Quotas().QuotaWithDefaults(defaults))
		}
	}
	return
}

func (r *PaasReconciler) UpdateClusterWideQuotaResources(
	ctx context.Context,
	quota *quotav1.ClusterResourceQuota,
) (changed bool, err error) {
	var allPaasResources paas_quota.QuotaLists
	if config, exists := getConfig().Capabilities[quota.ObjectMeta.Name]; !exists {
		return false, fmt.Errorf("missing capability config for %s", quota.ObjectMeta.Name)
	} else if !config.QuotaSettings.Clusterwide {
		return false, fmt.Errorf("running UpdateClusterWideQuota for non-clusterwide quota %s", quota.ObjectMeta.Name)
	} else if allPaasResources, err = r.FetchAllPaasCapabilityResources(ctx, quota, config.QuotaSettings.DefQuota); err != nil {
		return false, err
	} else {
		quota.Spec.Quota.Hard = corev1.ResourceList(allPaasResources.OptimalValues(
			config.QuotaSettings.Ratio,
			paas_quota.NewQuota(config.QuotaSettings.MinQuotas),
			paas_quota.NewQuota(config.QuotaSettings.MaxQuotas),
		))
	}

	return true, nil
}

// backendQuota is a code for Creating Quota
func backendClusterWideQuota(
	quotaName string,
	hardQuotas map[corev1.ResourceName]resourcev1.Quantity,
) *quotav1.ClusterResourceQuota {
	//matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}
	quota := &quotav1.ClusterResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterResourceQuota",
			APIVersion: "quota.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: quotaName,
		},
		Spec: quotav1.ClusterResourceQuotaSpec{
			Selector: quotav1.ClusterResourceQuotaSelector{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						getConfig().QuotaLabel: quotaName},
				},
			},
			Quota: corev1.ResourceQuotaSpec{
				Hard: hardQuotas,
			},
		},
	}
	return quota
}

func (r *PaasReconciler) RegisterClusterWideQuotas() {
	// get
	// generate if get fails
	// register owner
	// update quota values
}

func (r *PaasReconciler) UnRegisterClusterWideQuotas() {
	// get, exit if get fails
	// remove owner
	// remove clusterquote if no woner left
	// update quota values
	// save to k8s
}

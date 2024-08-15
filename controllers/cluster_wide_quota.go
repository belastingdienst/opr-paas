/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	paas_quota "github.com/belastingdienst/opr-paas/internal/quota"
	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	cwqPrefix string = "paas-"
)

func (r *PaasReconciler) FetchAllPaasCapabilityResources(
	ctx context.Context,
	quota *quotav1.ClusterResourceQuota,
	defaults map[string]string,
) (resources paas_quota.QuotaLists, err error) {
	capabilityName, err := ClusterWideCapabilityName(quota.Name)
	if err != nil {
		return
	}
	paas := &v1alpha1.Paas{}
	resources = paas_quota.NewQuotaLists()
	for _, reference := range quota.OwnerReferences {
		paasNamespacedName := types.NamespacedName{Name: reference.Name}
		if reference.Kind != "Paas" || reference.APIVersion != v1alpha1.GroupVersion.String() {
			// We don't bother the owner reference to a different CR is here.
			// We just don't add it to the list of resources.
			continue
		}
		if getErr := r.Get(ctx, paasNamespacedName, paas); getErr != nil {
			if errors.IsNotFound(getErr) {
				// Quota referencing a missing PaaS, no problem.
				continue
			}
			err = fmt.Errorf("Error occurring while retrieving the PaaS %s", getErr.Error())
			return
		}
		if paasCap, exists := paas.Spec.Capabilities.AsMap()[capabilityName]; !exists {
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
) (err error) {
	var allPaasResources paas_quota.QuotaLists
	if capabilityName, err := ClusterWideCapabilityName(quota.ObjectMeta.Name); err != nil {
		return err
	} else if config, exists := getConfig().Capabilities[capabilityName]; !exists {
		return fmt.Errorf("missing capability config for %s", capabilityName)
	} else if !config.QuotaSettings.Clusterwide {
		return fmt.Errorf("running UpdateClusterWideQuota for non-clusterwide quota %s", quota.ObjectMeta.Name)
	} else if allPaasResources, err = r.FetchAllPaasCapabilityResources(ctx, quota, config.QuotaSettings.DefQuota); err != nil {
		return err
	} else {
		quota.Spec.Quota.Hard = corev1.ResourceList(allPaasResources.OptimalValues(
			config.QuotaSettings.Ratio,
			paas_quota.NewQuota(config.QuotaSettings.MinQuotas),
			paas_quota.NewQuota(config.QuotaSettings.MaxQuotas),
		))
		return nil
	}
}

// backendQuota is a code for Creating Quota
func backendClusterWideQuota(
	quotaName string,
	hardQuotas map[corev1.ResourceName]resourcev1.Quantity,
) *quotav1.ClusterResourceQuota {
	// matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}
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
						getConfig().QuotaLabel: quotaName,
					},
				},
			},
			Quota: corev1.ResourceQuotaSpec{
				Hard: hardQuotas,
			},
		},
	}
	return quota
}

func ClusterWideQuotaName(capabilityName string) string {
	return fmt.Sprintf("%s%s", cwqPrefix, capabilityName)
}

func ClusterWideCapabilityName(quotaName string) (capabilityName string, err error) {
	var found bool
	if capabilityName, found = strings.CutPrefix(quotaName, cwqPrefix); !found {
		err = fmt.Errorf("failed to remove prefix")
	}
	return
}

func (r *PaasReconciler) FinalizeClusterWideQuotas(ctx context.Context, paas *v1alpha1.Paas) error {
	for capabilityName, capability := range paas.Spec.Capabilities.AsMap() {
		if capability.IsEnabled() {
			err := r.removeFromClusterWideQuota(ctx, paas, capabilityName)
			if err != nil && errors.IsNotFound(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func (r *PaasReconciler) ReconcileClusterWideQuota(ctx context.Context, paas *v1alpha1.Paas) error {
	for capabilityName, capability := range paas.Spec.Capabilities.AsMap() {
		if capability.IsEnabled() {
			err := r.addToClusterWideQuota(ctx, paas, capabilityName)
			if err != nil && errors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
		} else {
			err := r.removeFromClusterWideQuota(ctx, paas, capabilityName)
			if err != nil && errors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *PaasReconciler) addToClusterWideQuota(ctx context.Context, paas *v1alpha1.Paas, capabilityName string) error {
	var quota *quotav1.ClusterResourceQuota
	var exists bool
	quotaName := ClusterWideQuotaName(capabilityName)
	if config, exists := getConfig().Capabilities[capabilityName]; !exists {
		return fmt.Errorf("capability %s does not seem to exist in configuration", capabilityName)
	} else if !config.QuotaSettings.Clusterwide {
		return nil
	} else {
		quota = backendClusterWideQuota(quotaName,
			paas_quota.NewQuota(config.QuotaSettings.MinQuotas))
	}

	err := r.Get(ctx, types.NamespacedName{Name: quotaName}, quota)
	if err != nil && !errors.IsNotFound(err) {
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, quota, err.Error())
		return err
	}
	exists = (err == nil)

	if !paas.AmIOwner(quota.OwnerReferences) {
		controllerutil.SetOwnerReference(paas, quota, r.Scheme)
	}
	if err := r.UpdateClusterWideQuotaResources(ctx, quota); err != nil {
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusUpdate, quota, err.Error())
		return err
	}
	if exists {
		if err = r.Update(ctx, quota); err != nil {
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusUpdate, quota, err.Error())
			return err
		}
	} else {
		if err = r.Create(ctx, quota); err != nil {
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, quota, err.Error())
			return err
		}
	}
	paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, quota, "succeeded")
	return nil
}

func (r *PaasReconciler) removeFromClusterWideQuota(ctx context.Context, paas *v1alpha1.Paas, capabilityName string) error {
	var quota *quotav1.ClusterResourceQuota
	quotaName := fmt.Sprintf("%s%s", cwqPrefix, capabilityName)
	var capConfig config.ConfigCapability
	var exists bool
	if capConfig, exists = getConfig().Capabilities[capabilityName]; !exists {
		return fmt.Errorf("capability %s does not seem to exist", quotaName)
	} else {
		quota = backendClusterWideQuota(quotaName,
			paas_quota.NewQuota(capConfig.QuotaSettings.MinQuotas))
	}
	err := r.Get(ctx, types.NamespacedName{
		Name: quotaName,
	}, quota)
	if err != nil && errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, quota, err.Error())
		return err
	} else if !capConfig.QuotaSettings.Clusterwide {
		err := r.Delete(ctx, quota)
		if err != nil {
			if !errors.IsNotFound(err) {
				paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusDelete, quota, err.Error())
			}
			return err
		}
		return err
	} else if quota == nil {
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, quota, "unexpectedly quota is nil")
		return fmt.Errorf("unexpectedly quota %s is nil", quotaName)
	}
	quota.OwnerReferences = paas.WithoutMe(quota.OwnerReferences)
	if len(quota.OwnerReferences) < 1 {
		if err = r.Delete(ctx, quota); err != nil {
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusDelete, quota, err.Error())
			return err
		}
		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusDelete, quota, "succeeded")
		return nil
	}
	if err := r.UpdateClusterWideQuotaResources(ctx, quota); err != nil {
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusUpdate, quota, err.Error())
		return err
	} else if err = r.Update(ctx, quota); err != nil {
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusUpdate, quota, err.Error())
		return err
	}
	paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, quota, "succeeded")
	return nil
}
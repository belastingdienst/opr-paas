/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/belastingdienst/opr-paas/v3/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	paasquota "github.com/belastingdienst/opr-paas/v3/pkg/quota"
	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	cwqPrefix string = "paas"
)

func (r *PaasReconciler) fetchAllPaasCapabilityResources(
	ctx context.Context,
	quota *quotav1.ClusterResourceQuota,
	defaults map[corev1.ResourceName]resourcev1.Quantity,
) (resources paasquota.Quotas, err error) {
	capabilityName, err := clusterWideCapabilityName(quota.Name)
	if err != nil {
		return resources, err
	}
	paas := &v1alpha2.Paas{}
	resources = paasquota.NewQuotas()
	for _, reference := range quota.OwnerReferences {
		paasNamespacedName := types.NamespacedName{Name: reference.Name}
		if reference.Kind != "Paas" || reference.APIVersion != v1alpha2.GroupVersion.String() {
			// We don't bother the owner reference to a different CR is here.
			// We just don't add it to the list of resources.
			continue
		}
		if getErr := r.Get(ctx, paasNamespacedName, paas); getErr != nil {
			if k8serrors.IsNotFound(getErr) {
				// Quota referencing a missing Paas, no problem.
				continue
			}
			err = fmt.Errorf("error occurring while retrieving the Paas %s", getErr.Error())
			return resources, err
		}
		if paasCap, exists := paas.Spec.Capabilities[capabilityName]; !exists {
			resources.Append(defaults)
		} else {
			resources.Append(paasCap.Quotas().MergeWith(defaults))
		}
	}
	return resources, err
}

func (r *PaasReconciler) updateClusterWideQuotaResources(
	ctx context.Context,
	quota *quotav1.ClusterResourceQuota,
) (err error) {
	var allPaasResources paasquota.Quotas
	capabilityName, err := clusterWideCapabilityName(quota.Name)
	if err != nil {
		return err
	}
	myConfig, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return err
	}
	c, exists := myConfig.Spec.Capabilities[capabilityName]
	if !exists {
		return fmt.Errorf("missing capability config for %s", capabilityName)
	}
	if !c.QuotaSettings.Clusterwide {
		return fmt.Errorf("running UpdateClusterWideQuota for non-clusterwide quota %s", quota.Name)
	}
	allPaasResources, err = r.fetchAllPaasCapabilityResources(ctx,
		quota,
		c.QuotaSettings.DefQuota,
	)
	if err != nil {
		return err
	}
	quota.Spec.Quota.Hard = corev1.ResourceList(allPaasResources.OptimalValues(
		c.QuotaSettings.Ratio,
		c.QuotaSettings.MinQuotas,
		c.QuotaSettings.MaxQuotas,
	))
	return nil
}

// backendQuota is a code for Creating Quota
func backendClusterWideQuota(
	quotaName string,
	hardQuotas map[corev1.ResourceName]resourcev1.Quantity,
	quotaLabel string,
) *quotav1.ClusterResourceQuota {
	quota := &quotav1.ClusterResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: quotaName,
		},
		Spec: quotav1.ClusterResourceQuotaSpec{
			Selector: quotav1.ClusterResourceQuotaSelector{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						quotaLabel: quotaName,
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

func clusterWideQuotaName(capabilityName string) string {
	return join(cwqPrefix, capabilityName)
}

func clusterWideCapabilityName(quotaName string) (capabilityName string, err error) {
	var found bool
	if capabilityName, found = strings.CutPrefix(quotaName, cwqPrefix+"-"); !found {
		err = errors.New("failed to remove prefix")
	}
	return capabilityName, err
}

func (r *PaasReconciler) finalizeClusterWideQuotas(ctx context.Context, paas *v1alpha2.Paas) error {
	for capabilityName := range paas.Spec.Capabilities {
		err := r.removeFromClusterWideQuota(ctx, paas, capabilityName)
		if err != nil && k8serrors.IsNotFound(err) {
			continue
		}
		return err
	}
	return nil
}

func (r *PaasReconciler) reconcileClusterWideQuota(ctx context.Context, paas *v1alpha2.Paas) error {
	myconfig, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return err
	}

	for capabilityName := range myconfig.Spec.Capabilities {
		if _, exists := paas.Spec.Capabilities[capabilityName]; exists {
			err = r.addToClusterWideQuota(ctx, paas, capabilityName)
			if err != nil && k8serrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
		} else {
			err = r.removeFromClusterWideQuota(ctx, paas, capabilityName)
			if err != nil && k8serrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// needsUpdate compares the current and desired ClusterResourceQuota
func (r *PaasReconciler) needsUpdate(current, desired *quotav1.ClusterResourceQuota,
	paas *v1alpha2.Paas) (bool, error) {
	changed := false

	// Owner reference
	if !paas.AmIOwner(current.OwnerReferences) {
		if err := controllerutil.SetOwnerReference(paas, current, r.Scheme); err != nil {
			return false, err
		}
		changed = true
	}

	// Labels
	if !reflect.DeepEqual(current.Labels, desired.Labels) {
		current.Labels = desired.Labels
		changed = true
	}

	// Selector
	if !reflect.DeepEqual(current.Spec.Selector, desired.Spec.Selector) {
		current.Spec.Selector = desired.Spec.Selector
		changed = true
	}

	// Quotas
	if !reflect.DeepEqual(current.Spec.Quota.Hard, desired.Spec.Quota.Hard) {
		current.Spec.Quota.Hard = desired.Spec.Quota.Hard
		changed = true
	}

	return changed, nil
}

func (r *PaasReconciler) addToClusterWideQuota(ctx context.Context, paas *v1alpha2.Paas, capabilityName string) error {
	quotaName := clusterWideQuotaName(capabilityName)
	myConfig, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return err
	}
	capConfig, exists := myConfig.Spec.Capabilities[capabilityName]
	if !exists {
		return fmt.Errorf("capability %s does not exist in configuration", capabilityName)
	}
	if !capConfig.QuotaSettings.Clusterwide {
		return nil
	}

	desired := backendClusterWideQuota(quotaName, capConfig.QuotaSettings.MinQuotas, myConfig.Spec.QuotaLabel)

	// Try to fetch existing quota
	current := &quotav1.ClusterResourceQuota{}
	err = r.Get(ctx, client.ObjectKeyFromObject(desired), current)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	// If not found → create it
	if k8serrors.IsNotFound(err) {
		if err = controllerutil.SetOwnerReference(paas, desired, r.Scheme); err != nil {
			return err
		}
		if err = r.updateClusterWideQuotaResources(ctx, desired); err != nil {
			return err
		}
		return r.Create(ctx, desired)
	}

	// Found → check if anything changed
	var changed bool
	changed, err = r.needsUpdate(current, desired, paas)
	if err != nil {
		return err
	}

	// Update if changed
	if changed {
		if err = r.updateClusterWideQuotaResources(ctx, current); err != nil {
			return err
		}
		return r.Update(ctx, current)
	}

	return nil
}

func (r *PaasReconciler) removeFromClusterWideQuota(
	ctx context.Context,
	paas *v1alpha2.Paas,
	capabilityName string,
) error {
	myConfig, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return err
	}
	var quota *quotav1.ClusterResourceQuota
	quotaName := clusterWideQuotaName(capabilityName)
	var capConfig v1alpha2.ConfigCapability
	var exists bool
	if capConfig, exists = myConfig.Spec.Capabilities[capabilityName]; !exists {
		// If a Paas was created with a capability that was nog yet configured, we should be able to delete it.
		// Returning an error would block deletion.
		return nil
	}
	quota = backendClusterWideQuota(quotaName,
		capConfig.QuotaSettings.MinQuotas, myConfig.Spec.QuotaLabel)
	err = r.Get(ctx, client.ObjectKeyFromObject(quota), quota)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	} else if !capConfig.QuotaSettings.Clusterwide {
		return r.Delete(ctx, quota)
	} else if quota == nil {
		return fmt.Errorf("unexpectedly quota %s is nil", quotaName)
	}
	quota.OwnerReferences = paas.WithoutMe(quota.OwnerReferences)
	if len(quota.OwnerReferences) < 1 {
		return r.Delete(ctx, quota)
	}
	if err = r.updateClusterWideQuotaResources(ctx, quota); err != nil {
		return err
	} else if err = r.Update(ctx, quota); err != nil {
		return err
	}
	return nil
}

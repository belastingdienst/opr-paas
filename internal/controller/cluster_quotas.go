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

	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"
	paasquota "github.com/belastingdienst/opr-paas/internal/quota"

	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *PaasReconciler) ensureQuota(
	ctx context.Context,
	quota *quotav1.ClusterResourceQuota,
) error {
	// See if quota already exists and create if it doesn't
	found := &quotav1.ClusterResourceQuota{}
	err := r.Get(ctx, types.NamespacedName{
		Name: quota.Name,
	}, found)
	if err != nil && k8serrors.IsNotFound(err) {
		// Create the quota
		if err = r.Create(ctx, quota); err != nil {
			// creating the quota failed
			return err
		}
		// creating the quota was successful
		return nil
	} else if err != nil {
		// Error that isn't due to the quota not existing
		return err
	}
	// Update the quota
	found.OwnerReferences = quota.OwnerReferences
	found.Spec = quota.Spec
	if err = r.Update(ctx, found); err != nil {
		// updating the quota failed
		return err
	}
	return nil
}

// backendQuota is a code for Creating Quota
func (r *PaasReconciler) backendQuota(
	ctx context.Context,
	paas *v1alpha2.Paas, suffix string,
	hardQuotas map[corev1.ResourceName]resourcev1.Quantity,
) *quotav1.ClusterResourceQuota {
	var quotaName string
	if suffix == "" {
		quotaName = paas.Name
	} else {
		quotaName = fmt.Sprintf("%s-%s", paas.Name, suffix)
	}
	_, logger := logging.GetLogComponent(ctx, "quota")
	logger.Info().Msg("defining quota")
	// matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}
	quota := &quotav1.ClusterResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterResourceQuota",
			APIVersion: "quota.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   quotaName,
			Labels: paas.ClonedLabels(),
		},
		Spec: quotav1.ClusterResourceQuotaSpec{
			Selector: quotav1.ClusterResourceQuotaSelector{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						config.GetConfig().Spec.QuotaLabel: quotaName,
					},
				},
			},
			Quota: corev1.ResourceQuotaSpec{
				Hard: hardQuotas,
			},
		},
	}

	logger.Info().Msg("setting owner")

	if err := controllerutil.SetControllerReference(paas, quota, r.Scheme); err != nil {
		logger.Err(err).Msg("error setting owner")
	}

	return quota
}

func (r *PaasReconciler) backendEnabledQuotas(
	ctx context.Context,
	paas *v1alpha2.Paas,
) (quotas []*quotav1.ClusterResourceQuota, err error) {
	paasConfigSpec := config.GetConfig().Spec
	quotas = append(quotas, r.backendQuota(ctx, paas, "", paas.Spec.Quota))
	for name, capability := range paas.Spec.Capabilities {
		if capConfig, exists := paasConfigSpec.Capabilities[name]; !exists {
			return nil, errors.New("a capability is requested, but not configured")
		} else if !capConfig.QuotaSettings.Clusterwide {
			defaults := capConfig.QuotaSettings.DefQuota
			quotaValues := capability.Quotas().MergeWith(defaults)
			quotas = append(quotas, r.backendQuota(ctx, paas, name, quotaValues))
		}
	}
	return quotas, nil
}

// PaasQuotas can hold a set of Quota's for a Paas (or PaasCapability)
type PaasQuotas map[string]paasquota.Quota

func (r *PaasReconciler) backendUnneededQuotas(
	paas *v1alpha2.Paas,
) (quotas []string) {
	paasConfigSpec := config.GetConfig().Spec
	for name, capConfig := range paasConfigSpec.Capabilities {
		if _, exists := paas.Spec.Capabilities[name]; !exists {
			quotas = append(quotas, join(paas.Name, name))
		} else if capConfig.QuotaSettings.Clusterwide {
			quotas = append(quotas, join(paas.Name, name))
		}
	}
	return quotas
}

func (r *PaasReconciler) finalizeClusterQuota(ctx context.Context, quotaName string) error {
	ctx, logger := logging.GetLogComponent(ctx, "quota")
	logger.Info().Msg("finalizing")
	obj := &quotav1.ClusterResourceQuota{}
	if err := r.Get(ctx, types.NamespacedName{
		Name: quotaName,
	}, obj); err != nil && k8serrors.IsNotFound(err) {
		logger.Info().Msg("does not exist")
		return nil
	} else if err != nil {
		logger.Err(err).Msg("error retrieving info")
		return err
	}
	logger.Info().Msg("deleting")
	return r.Delete(ctx, obj)
}

func (r *PaasReconciler) reconcileQuotas(
	ctx context.Context,
	paas *v1alpha2.Paas,
) (err error) {
	ctx, logger := logging.GetLogComponent(ctx, "quota")
	logger.Info().Msg("creating quotas for Paas")
	// Create quotas if needed
	quotas, err := r.backendEnabledQuotas(ctx, paas)
	if err != nil {
		return err
	}
	for _, q := range quotas {
		logger.Info().Msg("creating quota " + q.Name + " for PAAS object ")
		if err := r.ensureQuota(ctx, q); err != nil {
			logger.Err(err).Msgf("failure while creating quota %s", q.Name)
			return err
		}
	}

	for _, name := range r.backendUnneededQuotas(paas) {
		logger.Info().Msg("cleaning quota " + name + " for PAAS object ")
		if err := r.finalizeClusterQuota(ctx, name); err != nil {
			logger.Err(err).Msgf("failure while finalizing quota %s", name)
			return err
		}
	}

	return nil
}

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
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"
	paasquota "github.com/belastingdienst/opr-paas/internal/quota"

	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureQuota ensures Quota presence
func (r *PaasReconciler) EnsureQuota(
	ctx context.Context,
	paas *v1alpha1.Paas,
	quota *quotav1.ClusterResourceQuota,
) error {
	// See if quota already exists and create if it doesn't
	found := &quotav1.ClusterResourceQuota{}
	err := r.Get(ctx, types.NamespacedName{
		Name: quota.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {
		// Create the quota
		if err = r.Create(ctx, quota); err != nil {
			// creating the quota failed
			return err
		} else {
			// creating the quota was successful
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the quota not existing
		return err
	} else {
		// Update the quota
		found.OwnerReferences = quota.OwnerReferences
		found.Spec = quota.Spec
		if err = r.Update(ctx, found); err != nil {
			// updating the quota failed
			return err
		}
		return nil
	}
}

// backendQuota is a code for Creating Quota
func (r *PaasReconciler) backendQuota(
	ctx context.Context,
	paas *v1alpha1.Paas, suffix string,
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

func (r *PaasReconciler) BackendEnabledQuotas(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (quotas []*quotav1.ClusterResourceQuota, err error) {
	paasConfigSpec := config.GetConfig().Spec
	quotas = append(quotas, r.backendQuota(ctx, paas, "", paas.Spec.Quota))
	for name, cap := range paas.Spec.Capabilities {
		if capConfig, exists := paasConfigSpec.Capabilities[name]; !exists {
			return nil, fmt.Errorf("a capability is requested, but not configured")
		} else if cap.IsEnabled() {
			if !capConfig.QuotaSettings.Clusterwide {
				defaults := capConfig.QuotaSettings.DefQuota
				quotaValues := cap.Quotas().MergeWith(
					defaults)
				quotas = append(quotas,
					r.backendQuota(ctx, paas, name, quotaValues))
			}
		}
	}
	return quotas, nil
}

type PaasQuotas map[string]paasquota.Quota

func (r *PaasReconciler) BackendUnneededQuotas(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (quotas []string) {
	paasConfigSpec := config.GetConfig().Spec
	for name, cap := range paas.Spec.Capabilities {
		if capConfig, exists := paasConfigSpec.Capabilities[name]; !exists {
			quotas = append(quotas, fmt.Sprintf("%s-%s", paas.Name, name))
		} else if !cap.IsEnabled() || capConfig.QuotaSettings.Clusterwide {
			quotas = append(quotas, fmt.Sprintf("%s-%s", paas.Name, name))
		}
	}
	return quotas
}

func (r *PaasReconciler) FinalizeClusterQuota(ctx context.Context, paas *v1alpha1.Paas, quotaName string) error {
	ctx, logger := logging.GetLogComponent(ctx, "quota")
	logger.Info().Msg("finalizing")
	obj := &quotav1.ClusterResourceQuota{}
	if err := r.Get(ctx, types.NamespacedName{
		Name: quotaName,
	}, obj); err != nil && errors.IsNotFound(err) {
		logger.Info().Msg("does not exist")
		return nil
	} else if err != nil {
		logger.Err(err).Msg("error retrieving info")
		return err
	} else {
		logger.Info().Msg("deleting")
		return r.Delete(ctx, obj)
	}
}

func (r *PaasNSReconciler) FinalizeClusterQuota(ctx context.Context, paasns *v1alpha1.PaasNS) error {
	ctx, logger := logging.GetLogComponent(ctx, "quota")
	logger.Info().Msg("finalizing")
	obj := &quotav1.ClusterResourceQuota{}
	if err := r.Get(ctx, types.NamespacedName{
		Name: paasns.NamespaceName(),
	}, obj); err != nil && errors.IsNotFound(err) {
		logger.Info().Msg("does not exist")
		return nil
	} else if err != nil {
		logger.Err(err).Msg("error retrieving info")
		return err
	} else {
		logger.Info().Msg("deleting")
		return r.Delete(ctx, obj)
	}
}

func (r *PaasReconciler) FinalizeClusterQuotas(ctx context.Context, paas *v1alpha1.Paas) error {
	suffixes := []string{
		"",
	}
	for name := range paas.Spec.Capabilities {
		suffixes = append(suffixes, fmt.Sprintf("-%s", name))
	}

	var err error
	for _, suffix := range suffixes {
		quotaName := fmt.Sprintf("%s%s", paas.Name, suffix)
		if cleanErr := r.FinalizeClusterQuota(ctx, paas, quotaName); cleanErr != nil {
			err = cleanErr
		}
	}
	return err
}

func (r *PaasReconciler) ReconcileQuotas(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (err error) {
	ctx, logger := logging.GetLogComponent(ctx, "quota")
	logger.Info().Msg("creating quotas for Paas")
	// Create quotas if needed
	if quotas, err := r.BackendEnabledQuotas(ctx, paas); err != nil {
		return err
	} else {
		for _, q := range quotas {
			logger.Info().Msg("creating quota " + q.Name + " for PAAS object ")
			if err := r.EnsureQuota(ctx, paas, q); err != nil {
				logger.Err(err).Msgf("failure while creating quota %s", q.Name)
				return err
			}
		}
	}
	// TODO(portly-halicore-76) remove once quota is removed from Status in a future release
	paas.Status.Quota = map[string]paasquota.Quota{}
	for _, name := range r.BackendUnneededQuotas(ctx, paas) {
		logger.Info().Msg("cleaning quota " + name + " for PAAS object ")
		if err := r.FinalizeClusterQuota(ctx, paas, name); err != nil {
			logger.Err(err).Msgf("failure while finalizing quota %s", name)
			return err
		}
	}

	return nil
}

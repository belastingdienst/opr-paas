/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"errors"
	"maps"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/config"
	"github.com/belastingdienst/opr-paas/v3/internal/logging"
	"github.com/belastingdienst/opr-paas/v3/internal/templating"
	paasquota "github.com/belastingdienst/opr-paas/v3/pkg/quota"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	err := r.Get(ctx, client.ObjectKeyFromObject(quota), found)
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
) (*quotav1.ClusterResourceQuota, error) {
	var quotaName string
	if suffix == "" {
		quotaName = paas.Name
	} else {
		quotaName = join(paas.Name, suffix)
	}

	_, logger := logging.GetLogComponent(ctx, logging.ControllerClusterQuotaComponent)
	logger.Info().Msg("defining quota")

	labels := map[string]string{}
	myConfig, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return nil, err
	}
	labelTemplater := templating.NewTemplater(*paas, myConfig)
	for name, tpl := range myConfig.Spec.Templating.ClusterQuotaLabels {
		var result templating.TemplateResult
		result, err = labelTemplater.TemplateToMap(name, tpl)
		if err != nil {
			return nil, err
		}
		maps.Copy(labels, result)
	}

	// matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}
	quota := &quotav1.ClusterResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:   quotaName,
			Labels: labels,
		},
		Spec: quotav1.ClusterResourceQuotaSpec{
			Selector: quotav1.ClusterResourceQuotaSelector{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						myConfig.Spec.QuotaLabel: quotaName,
					},
				},
			},
			Quota: corev1.ResourceQuotaSpec{
				Hard: hardQuotas,
			},
		},
	}

	logger.Info().Msg("setting owner")

	if err = controllerutil.SetControllerReference(paas, quota, r.Scheme); err != nil {
		logger.Err(err).Msg("error setting owner")
	}

	return quota, nil
}

func (r *PaasReconciler) backendEnabledQuotas(
	ctx context.Context,
	paas *v1alpha2.Paas,
) (quotas []*quotav1.ClusterResourceQuota, err error) {
	myConfig, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return nil, err
	}

	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerClusterQuotaComponent)

	nsDefs, err := r.nsDefsFromPaas(ctx, paas)
	if err != nil {
		logger.Err(err).Msg("could not get nsDefs from paas")
		return nil, err
	}
	logger.Debug().Msgf("Need to manage resources for %d namespaces", len(nsDefs))

	// if there are paasNs resources or if there are namespaces defined in the paas spec,
	// we need a generic quota named after the paas
	for _, nsDef := range nsDefs {
		if nsDef.capName == "" {
			// if the nsdef isn't a capability, define a quota
			var quota *quotav1.ClusterResourceQuota
			quota, err = r.backendQuota(ctx, paas, "", paas.Spec.Quota)
			if err != nil {
				return nil, err
			}
			// add quota to the quota's definitions
			quotas = append(quotas, quota)
			break
		}
	}

	for name, capability := range paas.Spec.Capabilities {
		if capConfig, exists := myConfig.Spec.Capabilities[name]; !exists {
			return nil, errors.New("a capability is requested, but not configured")
		} else if !capConfig.QuotaSettings.Clusterwide {
			defaults := capConfig.QuotaSettings.DefQuota
			quotaValues := capability.Quotas().MergeWith(defaults)
			var capQuota *quotav1.ClusterResourceQuota
			capQuota, err = r.backendQuota(ctx, paas, name, quotaValues)
			if err != nil {
				return nil, err
			}
			quotas = append(quotas, capQuota)
		}
	}
	return quotas, nil
}

// PaasQuotas can hold a set of Quota's for a Paas (or PaasCapability)
type PaasQuotas map[string]paasquota.Quota

func (r *PaasReconciler) backendUnneededQuotas(ctx context.Context,
	paas *v1alpha2.Paas,
) (quotas []string, err error) {
	myConfig, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return nil, err
	}
	for name, capConfig := range myConfig.Spec.Capabilities {
		if _, exists := paas.Spec.Capabilities[name]; !exists {
			quotas = append(quotas, join(paas.Name, name))
		} else if capConfig.QuotaSettings.Clusterwide {
			quotas = append(quotas, join(paas.Name, name))
		}
	}
	return quotas, nil
}

func (r *PaasReconciler) finalizeClusterQuota(ctx context.Context, quotaName string) error {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerClusterQuotaComponent)
	logger.Info().Msg("finalizing")
	quota := &quotav1.ClusterResourceQuota{}
	if err := r.Get(ctx, types.NamespacedName{
		Name: quotaName,
	}, quota); err != nil && k8serrors.IsNotFound(err) {
		logger.Info().Msg("does not exist")
		return nil
	} else if err != nil {
		logger.Err(err).Msg("error retrieving info")
		return err
	}
	logger.Info().Msg("deleting")
	return r.Delete(ctx, quota)
}

func (r *PaasReconciler) reconcileQuotas(
	ctx context.Context,
	paas *v1alpha2.Paas,
) (err error) {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerClusterQuotaComponent)
	logger.Info().Msg("creating quotas for Paas")

	// Create quotas if needed
	quotas, err := r.backendEnabledQuotas(ctx, paas)
	if err != nil {
		return err
	}
	for _, q := range quotas {
		logger.Info().Msg("creating quota " + q.Name + " for PAAS object ")
		if err = r.ensureQuota(ctx, q); err != nil {
			logger.Err(err).Msgf("failure while creating quota %s", q.Name)
			return err
		}
	}

	unneededQuotas, err := r.backendUnneededQuotas(ctx, paas)
	if err != nil {
		return err
	}
	for _, name := range unneededQuotas {
		logger.Info().Msg("cleaning quota " + name + " for PAAS object ")
		if err = r.finalizeClusterQuota(ctx, name); err != nil {
			logger.Err(err).Msgf("failure while finalizing quota %s", name)
			return err
		}
	}

	return nil
}

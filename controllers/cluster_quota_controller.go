package controllers

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ensureQuota ensures Quota presence
func (r *PaasReconciler) EnsureQuota(
	ctx context.Context,
	request reconcile.Request,
	quota *quotav1.ClusterResourceQuota,
) error {
	// See if quota already exists and create if it doesn't
	found := &quotav1.ClusterResourceQuota{}
	err := r.Get(ctx, types.NamespacedName{
		Name: quota.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the quota
		err = r.Create(ctx, quota)

		if err != nil {
			// creating the quota failed
			return err
		} else {
			// creating the quota was successful
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the quota not existing
		return err
	}

	return nil
}

// backendQuota is a code for Creating Quota
func (r *PaasReconciler) backendQuota(
	ctx context.Context,
	paas *v1alpha1.Paas, suffix string,
	hardQuotas map[corev1.ResourceName]resourcev1.Quantity,
) *quotav1.ClusterResourceQuota {
	var quotaName string
	if suffix == "" {
		quotaName = paas.ObjectMeta.Name
	} else {
		quotaName = fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, suffix)
	}
	logger := getLogger(ctx, paas, "Quota", quotaName)
	logger.Info("Defining quota")
	//matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}
	quota := &quotav1.ClusterResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterResourceQuota",
			APIVersion: "quota.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   quotaName,
			Labels: paas.Labels,
		},
		Spec: quotav1.ClusterResourceQuotaSpec{
			Selector: quotav1.ClusterResourceQuotaSelector{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"clusterquotagroup": quotaName},
				},
			},
			Quota: corev1.ResourceQuotaSpec{
				Hard: hardQuotas,
			},
		},
	}

	logger.Info("Setting owner")
	controllerutil.SetControllerReference(paas, quota, r.Scheme)
	return quota
}

func (r *PaasReconciler) BackendQuotas(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (quotas []*quotav1.ClusterResourceQuota) {
	quotas = append(quotas, r.backendQuota(ctx, paas, "", paas.Spec.Quota))
	if paas.Spec.Capabilities.ArgoCD.Enabled {
		quotas = append(quotas, r.backendQuota(ctx, paas, "argocd", paas.Spec.Capabilities.ArgoCD.QuotaWithDefaults()))
	}
	if paas.Spec.Capabilities.CI.Enabled {
		quotas = append(quotas, r.backendQuota(ctx, paas, "ci", paas.Spec.Capabilities.CI.QuotaWithDefaults()))
	}
	if paas.Spec.Capabilities.Grafana.Enabled {
		quotas = append(quotas, r.backendQuota(ctx, paas, "grafana", paas.Spec.Capabilities.Grafana.QuotaWithDefaults()))
	}
	if paas.Spec.Capabilities.SSO.Enabled {
		quotas = append(quotas, r.backendQuota(ctx, paas, "sso", paas.Spec.Capabilities.SSO.QuotaWithDefaults()))
	}

	return quotas
}

func (r *PaasReconciler) finalizeClusterQuota(ctx context.Context, paas *v1alpha1.Paas, quotaName string) error {
	logger := getLogger(ctx, paas, "Quota", quotaName)
	logger.Info("Finalizing")
	obj := &quotav1.ClusterResourceQuota{}
	if err := r.Get(ctx, types.NamespacedName{
		Name: quotaName,
	}, obj); err != nil && errors.IsNotFound(err) {
		logger.Info("Does not exist")
		return nil
	} else if err != nil {
		logger.Info("Error retrieving info: " + err.Error())
		return err
	} else {
		logger.Info("Deleting")
		return r.Delete(ctx, obj)
	}
}

func (r *PaasReconciler) FinalizeClusterQuotas(ctx context.Context, paas *v1alpha1.Paas) error {
	suffixes := []string{
		"",
		"-argocd",
		"-ci",
		"-grafana",
		"-sso",
	}
	var err error
	for _, suffix := range suffixes {
		quotaName := fmt.Sprintf("%s%s", paas.Name, suffix)
		if cleanErr := r.finalizeClusterQuota(ctx, paas, quotaName); cleanErr != nil {
			err = cleanErr
		}
	}
	return err
}
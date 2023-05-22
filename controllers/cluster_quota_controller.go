package controllers

import (
	"context"
	"fmt"

	mydomainv1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"

	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func labels(v *mydomainv1alpha1.Paas, tier string) map[string]string {
	// Fetches and sets labels

	return map[string]string{
		"app":             "visitors",
		"visitorssite_cr": v.Name,
		"tier":            tier,
	}
}

// ensureQuota ensures Quota presence
func (r *PaasReconciler) ensureQuota(request reconcile.Request,
	instance *mydomainv1alpha1.Paas,
	quota *quotav1.ClusterResourceQuota,
) (*reconcile.Result, error) {

	// See if quota already exists and create if it doesn't
	found := &quotav1.ClusterResourceQuota{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Name: quota.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the quota
		err = r.Create(context.TODO(), quota)

		if err != nil {
			// creating the quota failed
			return &reconcile.Result{}, err
		} else {
			// creating the quota was successful
			return nil, nil
		}
	} else if err != nil {
		// Error that isn't due to the quota not existing
		return &reconcile.Result{}, err
	}

	return nil, nil
}

// backendQuota is a code for Creating Quota
func (r *PaasReconciler) backendQuota(
	paas *mydomainv1alpha1.Paas, suffix string,
	hardQuotas map[corev1.ResourceName]resourcev1.Quantity,
) *quotav1.ClusterResourceQuota {
	var quotaName string
	if suffix == "" {
		quotaName = paas.ObjectMeta.Name
	} else {
		quotaName = fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, suffix)
	}
	//matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}
	quota := &quotav1.ClusterResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterResourceQuota",
			APIVersion: "quota.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      quotaName,
			Namespace: paas.Namespace,
			Labels:    paas.Labels,
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

	controllerutil.SetControllerReference(paas, quota, r.Scheme)
	return quota
}

func (r *PaasReconciler) backendQuotas(paas *mydomainv1alpha1.Paas) (quotas []*quotav1.ClusterResourceQuota) {
	quotas = append(quotas, r.backendQuota(paas, "", paas.Spec.Quota))
	if paas.Spec.Capabilities.ArgoCD.Enabled {
		quotas = append(quotas, r.backendQuota(paas, "argocd", paas.Spec.Capabilities.ArgoCD.QuotaWithDefaults()))
	}
	if paas.Spec.Capabilities.CI.Enabled {
		quotas = append(quotas, r.backendQuota(paas, "ci", paas.Spec.Capabilities.CI.QuotaWithDefaults()))
	}
	if paas.Spec.Capabilities.Grafana.Enabled {
		quotas = append(quotas, r.backendQuota(paas, "grafana", paas.Spec.Capabilities.Grafana.QuotaWithDefaults()))
	}
	if paas.Spec.Capabilities.SSO.Enabled {
		quotas = append(quotas, r.backendQuota(paas, "sso", paas.Spec.Capabilities.SSO.QuotaWithDefaults()))
	}
	return quotas
}

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

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureNamespace ensures Namespace presence in given namespace.
func EnsureNamespace(
	ctx context.Context,
	r client.Client,
	paas *v1alpha1.Paas,
	ns *corev1.Namespace,
	scheme *runtime.Scheme,
) error {
	// See if namespace exists and create if it doesn't
	found := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{
		Name: ns.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, ns)
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		if err := controllerutil.SetControllerReference(paas, found, scheme); err != nil {
			return err
		}
	}
	var changed bool
	for key, value := range ns.Labels {
		if orgValue, exists := found.Labels[key]; !exists || orgValue != value {
			changed = true
			found.Labels[key] = value
		}
	}
	if changed {
		return r.Update(ctx, found)
	}
	return nil
}

// backendNamespace is a code for Creating Namespace
func BackendNamespace(
	ctx context.Context,
	paas *v1alpha1.Paas,
	name string,
	quota string,
	scheme *runtime.Scheme,
) (*corev1.Namespace, error) {
	ctx, _ = logging.GetLogComponent(ctx, "namespace")
	logger := log.Ctx(ctx)
	logger.Info().Msgf("defining %s Namespace", name)
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: paas.ClonedLabels(),
		},
		Spec: corev1.NamespaceSpec{},
	}
	logger.Info().Msgf("setting Quotagroup %s", quota)
	ns.Labels[config.GetConfig().Spec.QuotaLabel] = quota

	argoNameSpace := fmt.Sprintf("%s-%s", paas.ManagedByPaas(), config.GetConfig().Spec.ManagedBySuffix)
	logger.Info().Msg("setting managed_by_label")
	ns.Labels[config.GetConfig().Spec.ManagedByLabel] = argoNameSpace

	logger.Info().Msg("setting requestor_label")
	ns.Labels[config.GetConfig().Spec.RequestorLabel] = paas.Spec.Requestor

	logger.Info().Str("Paas", paas.Name).Str("namespace", ns.Name).Msg("setting Owner")
	if err := controllerutil.SetControllerReference(paas, ns, scheme); err != nil {
		logger.Err(err).Msg("setControllerReference failure")
		return nil, err
	}
	for _, ref := range ns.OwnerReferences {
		logger.Info().Str("namespace", ns.Name).Str("reference", ref.Name).Msg("ownerReferences")
	}
	return ns, nil
}

func (r *PaasNSReconciler) FinalizeNamespace(
	ctx context.Context,
	paasns *v1alpha1.PaasNS,
	paas *v1alpha1.Paas,
) error {
	// Hoe voorkomen wij dat iemand een paasns maakt voor een verkeerde paas en als hij wordt weggegooid,
	// dat hij dan de verkeerde namespace weggooit???

	found := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{
		Name: paasns.NamespaceName(),
	}, found)
	if err != nil && errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		err = fmt.Errorf("cannot remove Namespace %s because Paas %s is not the owner", found.Name, paas.Name)
		return err
	} else if err = r.Delete(ctx, found); err != nil {
		// deleting the namespace failed
		return err
	} else {
		return nil
	}
}

func (r *PaasNSReconciler) ReconcileNamespaces(
	ctx context.Context,
	paas *v1alpha1.Paas,
	paasns *v1alpha1.PaasNS,
) (err error) {
	nsName := paasns.NamespaceName()
	var nsQuota string
	if configCapability, exists := config.GetConfig().Spec.Capabilities[paasns.Name]; !exists {
		nsQuota = paas.Name
	} else if !configCapability.QuotaSettings.Clusterwide {
		nsQuota = nsName
	} else {
		nsQuota = ClusterWideQuotaName(paasns.Name)
	}

	var ns *corev1.Namespace
	if ns, err = BackendNamespace(ctx, paas, nsName, nsQuota, r.Scheme); err != nil {
		return fmt.Errorf("failure while defining namespace %s: %s", nsName, err.Error())
	} else if err = EnsureNamespace(ctx, r.Client, paas, ns, r.Scheme); err != nil {
		return fmt.Errorf("failure while creating namespace %s: %s", nsName, err.Error())
	}
	return err
}

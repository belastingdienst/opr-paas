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
	r client.Client,
	ctx context.Context,
	addMessageFunc func(v1alpha1.PaasStatusLevel, v1alpha1.PaasStatusAction, client.Object, string),
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
		if err = r.Create(ctx, ns); err != nil {
			// creating the namespace failed
			addMessageFunc(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, ns, err.Error())
			return err
		} else {
			// creating the namespace was successful
			addMessageFunc(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, ns, "succeeded")
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		addMessageFunc(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, ns, err.Error())
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		addMessageFunc(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "updating owner")
		if err := controllerutil.SetControllerReference(paas, found, scheme); err != nil {
			return err
		}
	}
	var changed bool
	for key, value := range ns.ObjectMeta.Labels {
		if orgValue, exists := found.ObjectMeta.Labels[key]; !exists {
			addMessageFunc(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, fmt.Sprintf("adding label '%s'='%s'", key, value))
			// Not set yet
		} else if orgValue != value {
			addMessageFunc(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, fmt.Sprintf("updating label '%s'='%s'", key, value))
			// different
		} else {
			addMessageFunc(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, fmt.Sprintf("skipping label '%s'='%s'", key, value))
			// No action required
			continue
		}
		changed = true
		found.ObjectMeta.Labels[key] = value
	}
	if changed {
		addMessageFunc(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "updating namespace")
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
	setLogComponent(ctx, "namespace")
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
	ns.ObjectMeta.Labels[getConfig().Spec.QuotaLabel] = quota

	argoNameSpace := fmt.Sprintf("%s-argocd", paas.ManagedByPaas())
	logger.Info().Msg("setting managed_by_label")
	ns.ObjectMeta.Labels[getConfig().Spec.ManagedByLabel] = argoNameSpace

	logger.Info().Msg("setting requestor_label")
	ns.ObjectMeta.Labels[getConfig().Spec.RequestorLabel] = paas.Spec.Requestor

	logger.Info().Str("PaaS", paas.Name).Str("namespace", ns.Name).Msg("setting Owner")
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
	/*
	   Hoe voorkomen wij dat iemand een paasns maakt voor een verkeerde paas en als hij wordt weggegooid, dat hij dan de verkeerde namespace weggooit???
	*/

	found := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{
		Name: paasns.NamespaceName(),
	}, found)
	if err != nil && errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		err = fmt.Errorf("cannot remove Namespace %s because PaaS %s is not the owner", found.Name, paas.Name)
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
		return err
	} else if err = r.Delete(ctx, found); err != nil {
		// deleting the namespace failed
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusDelete, found, err.Error())
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
	if config, exists := GetConfig().Spec.Capabilities[paasns.Name]; !exists {
		nsQuota = paas.Name
	} else if !config.QuotaSettings.Clusterwide {
		nsQuota = nsName
	} else {
		nsQuota = ClusterWideQuotaName(paasns.Name)
	}

	var ns *corev1.Namespace
	if ns, err = BackendNamespace(ctx, paas, nsName, nsQuota, r.Scheme); err != nil {
		err = fmt.Errorf("failure while defining namespace %s: %s", nsName, err.Error())
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, paasns, err.Error())
	} else if err = EnsureNamespace(r.Client, ctx, paasns.Status.AddMessage, paas, ns, r.Scheme); err != nil {
		err = fmt.Errorf("failure while creating namespace %s: %s", nsName, err.Error())
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, ns, err.Error())
	}
	return
}

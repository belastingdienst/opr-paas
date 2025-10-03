/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"
	"maps"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/logging"
	"github.com/belastingdienst/opr-paas/v3/internal/templating"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureNamespace ensures Namespace presence in given namespace.
func (r *PaasReconciler) ensureNamespace(
	ctx context.Context,
	paas *v1alpha2.Paas,
	ns *corev1.Namespace,
) error {
	// See if namespace exists and create if it doesn't
	found := &corev1.Namespace{}
	err := r.Get(ctx, client.ObjectKeyFromObject(ns), found)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, ns)
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		if err = controllerutil.SetControllerReference(paas, found, r.Scheme); err != nil {
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

// backendNamespace is a code for defining Namespaces
func (r *PaasReconciler) backendNamespace(
	ctx context.Context,
	paas *v1alpha2.Paas,
	name string,
	quota string,
) (*corev1.Namespace, error) {
	_, logger := logging.GetLogComponent(ctx, logging.ControllerNamespaceComponent)
	logger.Info().Msgf("defining %s Namespace", name)

	labels := map[string]string{}
	myConfig, err := getConfigFromContext(ctx)
	if err != nil {
		return nil, err
	}
	labelTemplater := templating.NewTemplater(*paas, myConfig)
	for tplName, tpl := range myConfig.Spec.Templating.NamespaceLabels {
		var result templating.TemplateResult
		result, err = labelTemplater.TemplateToMap(tplName, tpl)
		if err != nil {
			return nil, err
		}
		maps.Copy(labels, result)
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: corev1.NamespaceSpec{},
	}
	logger.Info().Msgf("setting Quotagroup %s", quota)
	ns.Labels[myConfig.Spec.QuotaLabel] = quota
	ns.Labels[ManagedByLabelKey] = paas.Name

	logger.Info().Str("Paas", paas.Name).Str("namespace", ns.Name).Msg("setting Owner")
	if err := controllerutil.SetControllerReference(paas, ns, r.Scheme); err != nil {
		logger.Err(err).Msg("setControllerReference failure")
		return nil, err
	}
	for _, ref := range ns.OwnerReferences {
		logger.Info().Str("namespace", ns.Name).Str("reference", ref.Name).Msg("ownerReferences")
	}
	return ns, nil
}

func (r *PaasReconciler) reconcileNamespaces(
	ctx context.Context,
	paas *v1alpha2.Paas,
	nsDefs namespaceDefs,
) (err error) {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerNamespaceComponent)
	for _, nsDef := range nsDefs {
		var ns *corev1.Namespace
		if ns, err = r.backendNamespace(ctx, paas, nsDef.nsName, nsDef.quotaName); err != nil {
			return fmt.Errorf("failure while defining namespace %s: %s", nsDef.nsName, err.Error())
		} else if err = r.ensureNamespace(ctx, paas, ns); err != nil {
			return fmt.Errorf("failure while creating namespace %s: %s", nsDef.nsName, err.Error())
		}
		logger.Debug().Msgf("namespace %s successfully created with quotaName %s", nsDef.nsName, nsDef.quotaName)
	}
	return nil
}

// finalizeObsoleteNamespaces returns all groups owned by the specified Paas
func (r *PaasReconciler) finalizeObsoleteNamespaces(
	ctx context.Context,
	paas *v1alpha2.Paas,
	nsDefs namespaceDefs,
) (err error) {
	var nss corev1.NamespaceList
	var i int
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerNamespaceComponent)
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{ManagedByLabelKey: paas.Name}),
	}
	err = r.List(ctx, &nss, listOpts...)
	if err != nil {
		return err
	}
	for _, ns := range nss.Items {
		if _, exists := nsDefs[ns.Name]; !exists {
			err = r.Delete(ctx, &ns)
			if err != nil {
				return err
			}
			i++
		}
	}
	logger.Debug().Msgf("found %d existing namespaces owned by Paas %s", i, paas.Name)
	return nil
}

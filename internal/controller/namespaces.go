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

	"github.com/belastingdienst/opr-paas/v2/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v2/internal/config"
	"github.com/belastingdienst/opr-paas/v2/internal/logging"
	"github.com/belastingdienst/opr-paas/v2/internal/templating"

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
func ensureNamespace(
	ctx context.Context,
	r client.Client,
	paas *v1alpha2.Paas,
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
func backendNamespace(
	ctx context.Context,
	paas *v1alpha2.Paas,
	name string,
	quota string,
	scheme *runtime.Scheme,
) (*corev1.Namespace, error) {
	ctx, _ = logging.GetLogComponent(ctx, "namespace")
	logger := log.Ctx(ctx)
	logger.Info().Msgf("defining %s Namespace", name)

	labels := map[string]string{}
	myConfig := config.GetConfig()
	labelTemplater := templating.NewTemplater(*paas, myConfig)
	for tplName, tpl := range myConfig.Spec.ResourceLabels.NamespaceLabels {
		result, err := labelTemplater.TemplateToMap(tplName, tpl)
		if err != nil {
			return nil, err
		}
		maps.Copy(labels, result)
	}

	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: corev1.NamespaceSpec{},
	}
	logger.Info().Msgf("setting Quotagroup %s", quota)
	ns.Labels[config.GetConfig().Spec.QuotaLabel] = quota
	ns.Labels[ManagedByLabelKey] = paas.Name

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

func (r *PaasReconciler) reconcileNamespaces(
	ctx context.Context,
	paas *v1alpha2.Paas,
	nsDefs namespaceDefs,
) (err error) {
	ctx, logger := logging.GetLogComponent(ctx, "namespace")
	for _, nsDef := range nsDefs {
		var ns *corev1.Namespace
		if ns, err = backendNamespace(ctx, paas, nsDef.nsName, nsDef.quotaName, r.Scheme); err != nil {
			return fmt.Errorf("failure while defining namespace %s: %s", nsDef.nsName, err.Error())
		} else if err = ensureNamespace(ctx, r.Client, paas, ns, r.Scheme); err != nil {
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
	logger := log.Ctx(ctx)
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

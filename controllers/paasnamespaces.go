/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *PaasReconciler) GetPaasNs(ctx context.Context, paas *v1alpha1.Paas, name string,
	groups []string, secrets map[string]string) *v1alpha1.PaasNS {
	pns := &v1alpha1.PaasNS{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PaasNS",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: paas.Name,
			Labels:    paas.ClonedLabels(),
		},
		Spec: v1alpha1.PaasNSSpec{
			Paas:       paas.Name,
			Groups:     groups,
			SshSecrets: secrets,
		},
	}
	logger := getLogger(ctx, paas, pns.Kind, name)
	logger.Info("Defining")
	paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate,
		pns, "Setting requestor_label")
	pns.ObjectMeta.Labels[getConfig().RequestorLabel] = paas.Spec.Requestor

	logger.Info("Setting Owner")
	controllerutil.SetControllerReference(paas, pns, r.Scheme)
	return pns
}

func (r *PaasReconciler) EnsurePaasNs(ctx context.Context, paas *v1alpha1.Paas, pns *v1alpha1.PaasNS) error {
	logger := getLogger(ctx, paas, pns.Kind, pns.Name)
	logger.Info("Ensuring")

	// See if namespace exists and create if it doesn't
	found := &v1alpha1.PaasNS{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      pns.Name,
		Namespace: pns.Namespace,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		if err = r.Create(ctx, pns); err != nil {
			// creating the namespace failed
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, pns, err.Error())
			return err
		} else {
			// creating the namespace was successful
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, pns, "succeeded")
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, pns, err.Error())
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "updating owner")
		controllerutil.SetControllerReference(paas, found, r.Scheme)
	}
	// Since
	// var changed bool
	logger.Info("reconciling PaasNs", "PaasNs", pns, "found", found,
		"!group eq", !reflect.DeepEqual(pns.Spec.Groups, found.Spec.Groups),
		"!ssh eq", !reflect.DeepEqual(pns.Spec.SshSecrets, found.Spec.SshSecrets),
	)
	if pns.Spec.Paas != found.Spec.Paas {
		logger.Info("Paas changed", "PaasNs", pns)
		found.Spec.Paas = pns.Spec.Paas
		// changed = true
	}
	if len(pns.Spec.Groups) == 0 && len(found.Spec.Groups) == 0 {
		logger.Info("No groups defined", "PaasNs", pns)
	} else if !reflect.DeepEqual(pns.Spec.Groups, found.Spec.Groups) {
		logger.Info("Groups changed", "PaasNs", pns)
		found.Spec.Groups = pns.Spec.Groups
		// changed = true
	}
	if len(pns.Spec.Groups) == 0 && len(found.Spec.Groups) == 0 {
		logger.Info("no sshSecrets defined", "PaasNs", pns)
	} else if !reflect.DeepEqual(pns.Spec.SshSecrets, found.Spec.SshSecrets) {
		logger.Info("sshSecrets changed", "PaasNs", pns)
		found.Spec.SshSecrets = pns.Spec.SshSecrets
		// changed = true
	}
	for key, value := range pns.ObjectMeta.Labels {
		if orgValue, exists := found.ObjectMeta.Labels[key]; !exists {
			// Not set yet
		} else if orgValue != value {
			// different
		} else {
			// No action required
			continue
		}
		logger.Info(fmt.Sprintf("Label [%s] changed", key), "PaasNs", pns)
		// changed = true
		found.ObjectMeta.Labels[key] = value
	}
	// if changed {
	logger.Info("Updating PaasNs", "PaasNs", pns)
	return r.Update(ctx, found)
	// }
	// return nil
}

func (r *PaasReconciler) FinalizePaasNss(ctx context.Context, paas *v1alpha1.Paas) error {
	logger := getLogger(ctx, paas, "Namespace", "")
	logger.Info("Finalizing")

	enabledNs := paas.AllEnabledNamespaces()

	// Loop through all namespaces and remove when not should be
	pnsList := &v1alpha1.PaasNSList{}
	if err := r.List(ctx, pnsList, &client.ListOptions{Namespace: paas.Name}); err != nil {
		return err
	}

	for _, pns := range pnsList.Items {
		if !paas.AmIOwner(pns.OwnerReferences) {
			// logger.Info("Skipping finalization", "Namespace", ns.Name, "Reason", "I am not owner")
		} else if _, isEnabled := enabledNs[pns.Name]; isEnabled {
			// logger.Info("Skipping finalization", "Namespace", ns.Name, "Reason", "Should be there")
		} else if err := r.Delete(ctx, &pns); err != nil {
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusDelete, &pns, err.Error())
			// logger.Error(err, "Could not delete ns", "Namespace", ns.Name)
		} else {
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusDelete, &pns, "succeeded")
		}
	}
	return nil
}

func (r *PaasReconciler) ReconcilePaasNss(
	ctx context.Context,
	paas *v1alpha1.Paas,
	logger logr.Logger,
) error {
	logger.Info("Creating default namespace to hold PaasNs resources for PAAS object")
	if ns, err := BackendNamespace(ctx, paas, paas.Name, paas.Name, r.Scheme); err != nil {
		logger.Error(err, fmt.Sprintf("Failure while defining namespace %s", paas.Name))
		return err
	} else if err = EnsureNamespace(r.Client, ctx, paas.Status.AddMessage, paas, ns, r.Scheme); err != nil {
		logger.Error(err, fmt.Sprintf("Failure while creating namespace %s", paas.Name))
		return err
	} else {
		logger.Info("Creating PaasNs resources for PAAS object")
		for nsName := range paas.AllEnabledNamespaces() {
			pns := r.GetPaasNs(ctx, paas, nsName, paas.Spec.Groups.Names(), paas.GetNsSshSecrets(nsName))
			if err = r.EnsurePaasNs(ctx, paas, pns); err != nil {
				logger.Error(err, fmt.Sprintf("Failure while creating PaasNs %s",
					types.NamespacedName{Name: pns.Name, Namespace: pns.Namespace}))
				return err
			}
		}
	}
	logger.Info("Cleaning obsolete namespaces ")
	if err := r.FinalizePaasNss(ctx, paas); err != nil {
		return err
	}

	return nil
}

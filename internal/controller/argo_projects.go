/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	argo "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureAppProject ensures AppProject presence in given namespace.
func (r *PaasReconciler) EnsureAppProject(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	ctx = setLogComponent(ctx, "AppProject")
	log.Ctx(ctx).Info().Msg("Creating Argo Project")
	project, err := r.BackendAppProject(ctx, paas)
	if err != nil {
		return err
	}
	namespacedName := types.NamespacedName{
		Name:      project.Name,
		Namespace: project.Namespace,
	}

	// See if appProject exists and create if it doesn't
	found := &argo.AppProject{}
	err = r.Get(ctx, namespacedName, found)
	if err != nil && errors.IsNotFound(err) {
		// Create the namespace
		err = r.Create(ctx, project)

		if err != nil {
			// creating the appProject failed
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, found, err.Error())
			return err
		} else {
			// creating the appProject was successful
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, found, "succeeded")
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the appProject not existing
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "updating owner")
		if err := controllerutil.SetControllerReference(paas, found, r.Scheme); err != nil {
			return err
		}
		return r.Update(ctx, found)
	}
	return nil
}

// FinalizeAppProject finalizes AppProject
func (r *PaasReconciler) FinalizeAppProject(ctx context.Context, paas *v1alpha1.Paas) error {
	logger := log.Ctx(ctx)
	logger.Info().Msg("Finalizing App Project")
	appProject := &argo.AppProject{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      paas.Name,
		Namespace: getConfig().AppSetNamespace,
	}, appProject); err != nil && errors.IsNotFound(err) {
		logger.Info().Msg("App Project already deleted")
		return nil
	} else if err != nil {
		logger.Err(err).Msg("Error retrieving App Project")
		return err
	} else {
		logger.Info().Msg("Deleting App Project")
		return r.Delete(ctx, appProject)
	}
}

// backendAppProject is code for Creating AppProject
func (r *PaasReconciler) BackendAppProject(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (*argo.AppProject, error) {
	name := paas.Name
	logger := log.Ctx(ctx)
	logger.Info().Msgf("Defining %s AppProject", name)
	p := &argo.AppProject{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AppProject",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: getConfig().AppSetNamespace,
			Labels:    paas.ClonedLabels(),
			// Only removes appProject when apps no longer reference appProject
			Finalizers: []string{
				"resources-finalizer.argocd.argoproj.io",
			},
		},
		Spec: argo.AppProjectSpec{
			ClusterResourceWhitelist: []metav1.GroupKind{
				{Group: "*", Kind: "*"},
			},
			Destinations: []argo.ApplicationDestination{
				{Namespace: "*", Server: "*"},
			},
			SourceRepos: []string{
				"*",
			},
		},
	}

	logger.Info().Msg("Setting Owner")
	if err := controllerutil.SetControllerReference(paas, p, r.Scheme); err != nil {
		return nil, err
	}
	return p, nil
}

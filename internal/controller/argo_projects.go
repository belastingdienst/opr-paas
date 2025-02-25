/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"
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
	ctx, logger := logging.GetLogComponent(ctx, "appproject")
	logger.Info().Msg("creating Argo Project")
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
			return err
		}
		return nil
	} else if err != nil {
		// Error that isn't due to the appProject not existing
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		if err := controllerutil.SetControllerReference(paas, found, r.Scheme); err != nil {
			return err
		}
		return r.Update(ctx, found)
	}
	return nil
}

// backendAppProject is code for Creating AppProject
func (r *PaasReconciler) BackendAppProject(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (*argo.AppProject, error) {
	name := paas.Name
	logger := log.Ctx(ctx)
	logger.Info().Msgf("defining %s AppProject", name)
	p := &argo.AppProject{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AppProject",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: config.GetConfig().ClusterWideArgoCDNamespace,
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

	logger.Info().Msg("setting Owner")
	if err := controllerutil.SetControllerReference(paas, p, r.Scheme); err != nil {
		return nil, err
	}
	return p, nil
}

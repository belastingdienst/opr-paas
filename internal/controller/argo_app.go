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
	argo "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"

	"github.com/rs/zerolog/log"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	appName = "paas-bootstrap"
)

// ensureArgoApp ensures ArgoApp presence in given argo application.
func (r *PaasReconciler) EnsureArgoApp(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	ctx, logger := logging.GetLogComponent(ctx, "argoapp")
	namespace := fmt.Sprintf("%s-%s", paas.Name, "argocd")
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      appName,
	}

	// See if argo application exists and create if it doesn't
	found := &argo.Application{}
	argoApp, err := r.backendArgoApp(ctx, paas)
	if err != nil {
		return err
	} else if err := r.Get(ctx, namespacedName, found); err == nil {
		logger.Info().Msg("argo Application already exists, updating")
		patch := client.MergeFrom(found.DeepCopy())
		found.Spec = argoApp.Spec
		return r.Patch(ctx, found, patch)
	} else if !kerrors.IsNotFound(err) {
		logger.Err(err).Msg("could not retrieve info of Argo Application")
		return err
	}
	logger.Info().Msg("creating Argo Application")
	return r.Create(ctx, argoApp)
}

// backendArgoApp is code for creating a ArgoApp
func (r *PaasReconciler) backendArgoApp(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (*argo.Application, error) {
	logger := log.Ctx(ctx)
	logger.Info().Msgf("defining %s Argo Application", appName)

	namespace := fmt.Sprintf("%s-%s", paas.Name, "argocd")
	argoConfig := paas.Spec.Capabilities["argocd"]
	argoConfig.SetDefaults()
	fields, err := argoConfig.CapExtraFields(config.GetConfig().Capabilities["argocd"].CustomFields)
	if err != nil {
		return nil, err
	}
	app := &argo.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
			Labels:    paas.ClonedLabels(),
		},

		Spec: argo.ApplicationSpec{
			Destination: argo.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: namespace,
			},
			IgnoreDifferences: []argo.ResourceIgnoreDifferences{
				{
					Group:        "argoproj.io",
					JSONPointers: []string{"/spec/generators"},
					Kind:         "ApplicationSet",
					Name:         config.GetConfig().ExcludeAppSetName,
				},
			},
			Project: "default",
			Source: &argo.ApplicationSource{
				RepoURL:        fields["git_url"],
				Path:           fields["git_path"],
				TargetRevision: fields["git_revision"],
			},
			SyncPolicy: &argo.SyncPolicy{
				Automated: &argo.SyncPolicyAutomated{
					SelfHeal: true,
				},
				SyncOptions: []string{"RespectIgnoreDifferences=true"},
			},
		},
	}

	logger.Info().Msg("setting Owner")
	if err := controllerutil.SetControllerReference(paas, app, r.Scheme); err != nil {
		return app, err
	}
	return app, nil
}

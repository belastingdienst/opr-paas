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
	"github.com/belastingdienst/opr-paas/internal/log"
	argo "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	appName = "paas-bootstrap"
)

// ensureArgoApp ensures ArgoApp presence in given argo application.
func (r *PaasNSReconciler) EnsureArgoApp(
	ctx context.Context,
	paasns *v1alpha1.PaasNS,
	paas *v1alpha1.Paas,
) error {
	if paasns.Name != "argocd" {
		return nil
	}

	ctx, logger := log.WithAttributes(ctx, "ArgoApp", appName)
	namespacedName := types.NamespacedName{
		Namespace: paasns.NamespaceName(),
		Name:      appName,
	}

	// See if argo application exists and create if it doesn't
	found := &argo.Application{}
	if argoApp, err := r.backendArgoApp(ctx, paasns, paas); err != nil {
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusAction(v1alpha1.PaasStatusInfo), argoApp, err.Error())
		return err
	} else if err := r.Get(ctx, namespacedName, found); err == nil {
		logger.Info("Argo Application already exists, updating")
		patch := client.MergeFrom(found.DeepCopy())
		found.Spec = argoApp.Spec
		paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, found, "succeeded")
		return r.Patch(ctx, found, patch)
	} else if !errors.IsNotFound(err) {
		logger.Error(err, "Could not retrieve info of Argo Application")
		paasns.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusAction(v1alpha1.PaasStatusInfo), argoApp, err.Error())
		return err
	} else {
		logger.Info("Creating Argo Application")
		paasns.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, argoApp, "succeeded")
		return r.Create(ctx, argoApp)
	}
}

// backendArgoApp is code for creating a ArgoApp
func (r *PaasNSReconciler) backendArgoApp(
	ctx context.Context,
	paasns *v1alpha1.PaasNS,
	paas *v1alpha1.Paas,
) (*argo.Application, error) {
	logger := getLogger(ctx, paasns, "ArgoApp", appName)
	logger.Info(fmt.Sprintf("Defining %s Argo Application", appName))

	namespace := paasns.NamespaceName()
	argoConfig := paas.Spec.Capabilities.ArgoCD
	argoConfig.SetDefaults()
	app := &argo.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
			Labels:    paasns.ClonedLabels(),
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
					Name:         getConfig().ExcludeAppSetName,
				},
			},
			Project: "default",
			Source: &argo.ApplicationSource{
				RepoURL:        argoConfig.GitUrl,
				Path:           argoConfig.GitPath,
				TargetRevision: argoConfig.GitRevision,
			},
			SyncPolicy: &argo.SyncPolicy{
				Automated: &argo.SyncPolicyAutomated{
					SelfHeal: true,
				},
				SyncOptions: []string{"RespectIgnoreDifferences=true"},
			},
		},
	}

	logger.Info("Setting Owner")
	if err := controllerutil.SetControllerReference(paas, app, r.Scheme); err != nil {
		return app, err
	}
	return app, nil
}

func (r *PaasNSReconciler) FinalizeArgoApp(
	ctx context.Context,
	paasns *v1alpha1.PaasNS,
) error {
	namespacedName := types.NamespacedName{
		Namespace: paasns.NamespaceName(),
		Name:      appName,
	}
	logger := getLogger(ctx, paasns, "Application", namespacedName.String())
	logger.Info("Finalizing")
	obj := &argo.Application{}
	if err := r.Get(ctx, namespacedName, obj); err != nil && errors.IsNotFound(err) {
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

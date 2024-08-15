/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	argo "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ensureAppProject ensures AppProject presence in given namespace.
func (r *PaasReconciler) EnsureAppProject(
	ctx context.Context,
	paas *v1alpha1.Paas,
	logger logr.Logger,
) error {
	logger.Info("Creating Argo Project")
	project := r.BackendAppProject(ctx, paas)
	namespacedName := types.NamespacedName{
		Name:      project.Name,
		Namespace: project.Namespace,
	}

	// See if namespace exists and create if it doesn't
	found := &argo.AppProject{}
	err := r.Get(ctx, namespacedName, found)
	if err != nil && errors.IsNotFound(err) {
		// Create the namespace
		err = r.Create(ctx, project)

		if err != nil {
			// creating the namespace failed
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, found, err.Error())
			return err
		} else {
			// creating the namespace was successful
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, found, "succeeded")
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		return err
		// Ownerreference creeert een issue waarbij de appsetcontroller niet de app wegggooid omdat het app project niet meer bestaat
		//	} else if !paas.AmIOwner(found.OwnerReferences) {
		//		paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, found, "updating owner")
		//		controllerutil.SetControllerReference(paas, found, r.Scheme)
		//		return r.Update(ctx, found)
	}
	return nil
}

// backendAppProject is a code for Creating AppProject
func (r *PaasReconciler) BackendAppProject(
	ctx context.Context,
	paas *v1alpha1.Paas,
) *argo.AppProject {
	name := paas.Name
	logger := getLogger(ctx, paas, "AppProject", name)
	logger.Info(fmt.Sprintf("Defining %s AppProject", name))
	p := &argo.AppProject{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AppProject",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: getConfig().AppSetNamespace,
			Labels:    paas.ClonedLabels(),
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

	// logger.Info("Setting Owner")
	// controllerutil.SetControllerReference(paas, p, r.Scheme)
	return p
}

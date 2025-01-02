/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// TODO(portly-halicore-76): replace logger.
// nolint:unused
// log is for logging in this package.
var Paaslog = logf.Log.WithName("Paas-resource")

// SetupPaasWebhookWithManager registers the webhook for Paas in the manager.
func SetupPaasWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&Paas{}).
		WithValidator(&PaasCustomValidator{}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-cpet-belastingdienst-nl-v1alpha1-paas,mutating=false,failurePolicy=fail,sideEffects=None,groups=cpet.belastingdienst.nl,resources=paas,verbs=create;update,versions=v1alpha1,name=vpaas-v1alpha1.kb.io,admissionReviewVersions=v1

// PaasCustomValidator struct is responsible for validating the Paas resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
// +kubebuilder:object:generate=false
type PaasCustomValidator struct {
	// TODO(portly-halicore-76): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &PaasCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	paas, ok := obj.(*Paas)
	if !ok {
		return nil, fmt.Errorf("expected a Paas object but got %T", obj)
	}
	Paaslog.Info("Validation for Paas upon creation", "name", paas.GetName())

	// TODO(portly-halicore-76): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	paas, ok := newObj.(*Paas)
	if !ok {
		return nil, fmt.Errorf("expected a Paas object for the newObj but got %T", newObj)
	}
	Paaslog.Info("Validation for Paas upon update", "name", paas.GetName())

	// TODO(portly-halicore-76): fill in your validation logic upon object update.

	return nil, nil
}

// TODO(portly-halicore-76): determine whether this can be left out
// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	Paas, ok := obj.(*Paas)
	if !ok {
		return nil, fmt.Errorf("expected a Paas object but got %T", obj)
	}
	Paaslog.Info("Validation for Paas upon deletion", "name", Paas.GetName())

	// TODO(portly-halicore-76): fill in your validation logic upon object deletion.

	return nil, nil
}

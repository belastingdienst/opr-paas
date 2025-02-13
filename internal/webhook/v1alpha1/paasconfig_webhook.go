/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/logging"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupPaasConfigWebhookWithManager registers the webhook for PaasConfig in the manager.
func SetupPaasConfigWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1alpha1.PaasConfig{}).
		WithValidator(&PaasConfigCustomValidator{client: mgr.GetClient()}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// revive:disable-line
// +kubebuilder:webhook:path=/validate-cpet-belastingdienst-nl-v1alpha1-paasconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=cpet.belastingdienst.nl,resources=paasconfig,verbs=create;update,versions=v1alpha1,name=vpaasconfig-v1alpha1.kb.io,admissionReviewVersions=v1

// PaasConfigCustomValidator struct is responsible for validating the PaasConfig resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
// +kubebuilder:object:generate=false
type PaasConfigCustomValidator struct {
	client client.Client
}

var _ webhook.CustomValidator = &PaasConfigCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type PaasConfig.
func (v *PaasConfigCustomValidator) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (warn admission.Warnings, err error) {
	var allErrs field.ErrorList

	paasconfig, ok := obj.(*v1alpha1.PaasConfig)
	if !ok {
		return nil, fmt.Errorf("expected a PaasConfig object but got %T", obj)
	}

	_, logger := logging.SetWebhookLogger(ctx, paasconfig)

	logger.Info().Msgf("Validation for creation of PaasConfig %s", paasconfig.GetName())

	// Deny creation from secondary or more PaasConfig resources
	if flderr := validateNoPaasConfigExists(ctx, v.client); flderr != nil {
		allErrs = append(allErrs, flderr)
	}

	return warn, allErrs.ToAggregate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type PaasConfig.
func (v *PaasConfigCustomValidator) ValidateUpdate(
	ctx context.Context,
	oldObj, newObj runtime.Object,
) (admission.Warnings, error) {
	paasconfig, ok := newObj.(*v1alpha1.PaasConfig)
	if !ok {
		return nil, fmt.Errorf("expected a PaasConfig object for the newObj but got %T", newObj)
	}
	_, logger := logging.GetLogComponent(ctx, "paasconfig_webhook_validate_update")
	logger.Info().Msgf("Validation for update of PaasConfig %s", paasconfig.GetName())

	// TODO(portly-halicore-76): fill in your validation logic upon object update.

	return nil, nil
}

// TODO(portly-halicore-76): determine whether this can be left out
// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type PaasConfig.
func (v *PaasConfigCustomValidator) ValidateDelete(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	paasconfig, ok := obj.(*v1alpha1.PaasConfig)
	if !ok {
		return nil, fmt.Errorf("expected a PaasConfig object but got %T", obj)
	}
	_, logger := logging.GetLogComponent(ctx, "paasconfig_webhook_validate_update")
	logger.Info().Msgf("Validation for deletion of PaasConfig %s", paasconfig.GetName())

	// TODO(portly-halicore-76): fill in your validation logic upon object deletion.

	return nil, nil
}

func validateNoPaasConfigExists(ctx context.Context, client client.Client) *field.Error {
	var list v1alpha1.PaasConfigList

	ctx, logger := logging.GetLogComponent(ctx, "webhook_paasconfig_validateNoPaasConfigExists")

	if err := client.List(ctx, &list); err != nil {
		err = fmt.Errorf("failed to retrieve PaasConfigList: %w", err)
		logger.Error().Msg(err.Error())
		return field.InternalError(&field.Path{}, err)
	}

	if len(list.Items) > 0 {
		return field.Forbidden(&field.Path{}, "another PaasConfig resource already exists")
	}

	return nil
}

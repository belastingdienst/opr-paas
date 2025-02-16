/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/belastingdienst/opr-paas/internal/logging"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupPaasWebhookWithManager registers the webhook for Paas in the manager.
func SetupPaasWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1alpha1.Paas{}).
		WithValidator(&PaasCustomValidator{client: mgr.GetClient()}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// revive:disable-line
// +kubebuilder:webhook:path=/validate-cpet-belastingdienst-nl-v1alpha1-paas,mutating=false,failurePolicy=fail,sideEffects=None,groups=cpet.belastingdienst.nl,resources=paas,verbs=create;update,versions=v1alpha1,name=vpaas-v1alpha1.kb.io,admissionReviewVersions=v1

// PaasCustomValidator struct is responsible for validating the Paas resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
// +kubebuilder:object:generate=false
type PaasCustomValidator struct {
	client client.Client
}

var _ webhook.CustomValidator = &PaasCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	paas, ok := obj.(*v1alpha1.Paas)
	if !ok {
		return nil, fmt.Errorf("expected a Paas object but got %T", obj)
	}
	ctx, logger := logging.SetWebhookLogger(ctx, paas)
	logger.Info().Msg("starting validation webhook for creation")

	return v.validate(ctx, paas)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateUpdate(
	ctx context.Context,
	_, newObj runtime.Object,
) (admission.Warnings, error) {
	paas, ok := newObj.(*v1alpha1.Paas)
	if !ok {
		return nil, fmt.Errorf("expected a Paas object for the newObj but got %T", newObj)
	}
	ctx, logger := logging.SetWebhookLogger(ctx, paas)
	logger.Info().Msg("starting validation webhook for update")

	return v.validate(ctx, paas)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	paas, ok := obj.(*v1alpha1.Paas)
	if !ok {
		return nil, fmt.Errorf("expected a Paas object but got %T", obj)
	}
	_, logger := logging.SetWebhookLogger(ctx, paas)
	logger.Info().Msg("starting validation webhook for deletion")

	// No validation needed for deletion.

	return nil, nil
}

type paasSpecValidator func(context.Context, client.Client, v1alpha1.PaasConfigSpec, *v1alpha1.Paas) ([]*field.Error, error)

func (v *PaasCustomValidator) validate(ctx context.Context, paas *v1alpha1.Paas) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings []string
	conf := config.GetConfig()
	// Check for uninitialized config
	if conf.DecryptKeysSecret.Name == "" {
		return nil, apierrors.NewInternalError(fmt.Errorf("uninitialized PaasConfig"))
	}

	for _, val := range []paasSpecValidator{
		validateCaps,
		validateSecrets,
		validateCustomFields,
	} {
		if errs, err := val(ctx, v.client, conf, paas); err != nil {
			return nil, apierrors.NewInternalError(err)
		} else if errs != nil {
			allErrs = append(allErrs, errs...)
		}
	}

	warnings = append(warnings, v.validateGroups(paas.Spec.Groups)...)

	if len(allErrs) == 0 && len(warnings) == 0 {
		return nil, nil
	} else if len(allErrs) == 0 {
		return warnings, nil
	}
	return warnings, apierrors.NewInvalid(
		schema.GroupKind{Group: v1alpha1.GroupVersion.Group, Kind: "Paas"},
		paas.Name,
		allErrs,
	)
}

// validateCaps returns an error if any of the passed capabilities is not configured.
func validateCaps(ctx context.Context, client client.Client, conf v1alpha1.PaasConfigSpec, paas *v1alpha1.Paas) ([]*field.Error, error) {
	var errs []*field.Error

	for name := range paas.Spec.Capabilities {
		if _, ok := conf.Capabilities[name]; !ok {
			errs = append(errs, field.Invalid(
				field.NewPath("spec").Child("capabilities"),
				name,
				"capability not configured",
			))
		}
	}

	return errs, nil
}

func validateSecrets(ctx context.Context, client client.Client, conf v1alpha1.PaasConfigSpec, paas *v1alpha1.Paas) ([]*field.Error, error) {
	decryptRes := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{
		Name:      conf.DecryptKeysSecret.Name,
		Namespace: conf.DecryptKeysSecret.Namespace,
	}, decryptRes); err != nil {
		return nil, fmt.Errorf("could not retrieve decryption secret: %w", err)
	}

	// TODO(AxiomaticFixedChimpanzee): this function never errors, refactor to remove it from the signature
	keys, _ := crypt.NewPrivateKeysFromSecretData(decryptRes.Data)
	// TODO(AxiomaticFixedChimpanzee): can't error when passed path is empty, could also refactor this
	rsa, _ := crypt.NewCryptFromKeys(keys, "", paas.Name)

	var errs []*field.Error
	for name, secret := range paas.Spec.SSHSecrets {
		if _, err := rsa.Decrypt(secret); err != nil {
			errs = append(errs, field.Invalid(
				field.NewPath("spec").Child("sshSecrets"),
				name,
				fmt.Sprintf("cannot be decrypted: %s", err),
			))
		}
	}

	return errs, nil
}

// validateCustomFields ensures that for a given capability in the Paas:
//   - all custom fields are configured for that capability in the PaasConfig
//   - all custom fields pass regular expression validation as configured in the PaasConfig if present
//
// Returns an internal error if the validation regexp cannot be compiled.
func validateCustomFields(ctx context.Context, client client.Client, conf v1alpha1.PaasConfigSpec, paas *v1alpha1.Paas) ([]*field.Error, error) {
	var errs []*field.Error

	for cname, c := range paas.Spec.Capabilities {
		// validateCaps() has already ensured the capability configuration exists
		if _, err := c.CapExtraFields(config.GetConfig().Capabilities[cname].CustomFields); err != nil {
			errs = append(errs, field.Invalid(
				field.NewPath("spec").Child("capabilities").Key(cname),
				"custom_fields",
				err.Error(),
			))

			continue
		}
	}

	return errs, nil
}

// validateGroups returns a warning for any of the passed groups which contain both users and a query.
func (v *PaasCustomValidator) validateGroups(groups v1alpha1.PaasGroups) (warnings []string) {
	for key, grp := range groups {
		if len(grp.Query) > 0 && len(grp.Users) > 0 {
			warnings = append(warnings, fmt.Sprintf(
				"%s contains both users and query, the users will be ignored",
				field.NewPath("spec").Child("groups").Key(key),
			))
		}
	}

	return warnings
}

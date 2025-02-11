/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"
	"fmt"
	"regexp"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/logging"
	"github.com/belastingdienst/opr-paas/internal/validate"
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
func (v *PaasConfigCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warn admission.Warnings, err error) {
	var allErrs field.ErrorList

	paasconfig, ok := obj.(*v1alpha1.PaasConfig)
	if !ok {
		return nil, fmt.Errorf("expected a PaasConfig object but got %T", obj)
	}

	_, logger := logging.SetWebhookLogger(ctx, paasconfig)

	logger.Info().Msgf("validation for creation of PaasConfig %s", paasconfig.GetName())

	// Deny creation from secondary or more PaasConfig resources
	if flderr := validateNoPaasConfigExists(ctx, v.client); flderr != nil {
		allErrs = append(allErrs, flderr)
	}

	// Ensure all required fields and values are there
	if flderr := validatePaasConfig(ctx, paasconfig.Spec); flderr != nil {
		allErrs = append(allErrs, *flderr...)
	}

	// Ensure LDAP.Host is syntactically valid string, connection check is not done
	if valid, err := validate.Hostname(paasconfig.Spec.LDAP.Host); !valid {
		logger.Error().Msg(err.Error())
		path := field.NewPath("PaasConfig").Child("Spec").Child("LDAP")
		allErrs = append(allErrs, field.Invalid(path, paasconfig.Spec.LDAP.Host, err.Error()))
	}

	if flderr := validateCapabilities(ctx, paasconfig.Spec.Capabilities); flderr != nil {
		allErrs = append(allErrs, *flderr...)
	}

	return warn, allErrs.ToAggregate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type PaasConfig.
func (v *PaasConfigCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	paasconfig, ok := newObj.(*v1alpha1.PaasConfig)
	if !ok {
		return nil, fmt.Errorf("expected a PaasConfig object for the newObj but got %T", newObj)
	}
	_, logger := logging.GetLogComponent(ctx, "paasconfig_webhook_validate_update")
	logger.Info().Msgf("validation for update of PaasConfig %s", paasconfig.GetName())

	// TODO(portly-halicore-76): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type PaasConfig.
func (v *PaasConfigCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	paasconfig, ok := obj.(*v1alpha1.PaasConfig)
	if !ok {
		return nil, fmt.Errorf("expected a PaasConfig object but got %T", obj)
	}

	ctx, logger := logging.GetLogComponent(ctx, "webhook_paasconfig_validateUpdate")
	logger.Info().Msgf("Validation for deletion of PaasConfig %s", paasconfig.GetName())

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

func validatePaasConfig(ctx context.Context, config v1alpha1.PaasConfigSpec) *field.ErrorList {
	ctx, logger := logging.GetLogComponent(ctx, "webhook_paasconfig_validatePaasConfig")
	var allErrs field.ErrorList

	if config.DecryptKeysSecret.Name == "" {
		logger.Error().Msg("DecryptKeysSecret is required and must have name")
		allErrs = append(allErrs, field.Required(field.NewPath("DecryptKeysSecret").Child("Name"), "field is required"))
	}

	if config.DecryptKeysSecret.Namespace == "" {
		logger.Error().Msg("DecryptKeysSecret is required and must have namespace")
		allErrs = append(allErrs, field.Required(field.NewPath("DecryptKeysSecret").Child("Namespace"), "field is required"))
	}

	// TODO: remove once GroupSyncList is removed
	if config.GroupSyncList.Name == "" {
		logger.Error().Msg("GroupSyncList is required and must have name")
		allErrs = append(allErrs, field.Required(field.NewPath("GroupSyncList").Child("Name"), "field is required"))
	}

	// TODO: remove once GroupSyncList is removed
	if config.GroupSyncList.Namespace == "" {
		logger.Error().Msg("GroupSyncList is required and must have namespace")
		allErrs = append(allErrs, field.Required(field.NewPath("GroupSyncList").Child("Namespace"), "field is required"))
	}

	if len(config.ClusterWideArgoCDNamespace) < 1 {
		logger.Error().Msg("ClusterWideArgoCDNamespace is required and must have at least 1 character")
		allErrs = append(allErrs, field.Invalid(field.NewPath("ClusterWideArgoCDNamespace"), config.ClusterWideArgoCDNamespace, "field is required and must have at least 1 character"))
	}

	// TODO: remove once ExcludeAppSetName is removed
	if len(config.ExcludeAppSetName) < 1 {
		logger.Error().Msg("ExcludeAppSetName is required and must have at least 1 character")
		allErrs = append(allErrs, field.Invalid(field.NewPath("ExcludeAppSetName"), config.ExcludeAppSetName, "field is required and must have at least 1 character"))
	}

	return &allErrs
}

func validateCapabilities(ctx context.Context, caps v1alpha1.ConfigCapabilities) *field.ErrorList {
	ctx, logger := logging.GetLogComponent(ctx, "webhook_paasconfig_validatePaasConfig")
	var allErrs field.ErrorList

	for name, capability := range caps {
		// Ensure valid quotasettings
		allErrs = append(allErrs, *validateQuotaSettings(ctx, capability.QuotaSettings)...)

		// For our custom fields
		for fieldName, customField := range capability.CustomFields {
			// Can't set both Required and Default
			if customField.Required && customField.Default != "" {
				msg := "custom field has both Required and Default set, which is invalid"
				logger.Error().Msg(msg)
				allErrs = append(allErrs, field.Invalid(field.NewPath("ConfigCapability").Child("CustomField"), fieldName, msg))
			}

			if customField.Validation != "" {
				// Must have compilable regex
				if valid, err := validate.StringIsRegex(customField.Validation); !valid {
					msg := fmt.Sprintf("custom field '%s' in capability '%s' has an invalid regex pattern", fieldName, name)
					logger.Error().Msg(msg)
					allErrs = append(allErrs, field.Invalid(
						field.NewPath("ConfigCapability").Child("CustomField").Child("Validation"),
						fieldName,
						err.Error()))
				}

				// Default field must conform to regex validation
				if customField.Default != "" {
					if matched, err := regexp.Match(customField.Validation, []byte(customField.Default)); err != nil {
						msg := fmt.Errorf("could not validate value %s: %s", customField.Default, err.Error())
						logger.Error().Msg(msg.Error())
						allErrs = append(allErrs, field.InternalError(field.NewPath("ConfigCapability").Child("CustomField").Child("Default"), msg))
					} else if !matched {
						msg := fmt.Sprintf("invalid value %s (does not match %s)", customField.Default, customField.Validation)
						logger.Error().Msg(msg)
						allErrs = append(allErrs, field.Invalid(
							field.NewPath("ConfigCapability").Child("CustomField").Child("Default"),
							fieldName,
							msg))
					}
				}
			}
		}
	}

	return &allErrs
}

func validateQuotaSettings(ctx context.Context, qs v1alpha1.ConfigQuotaSettings) *field.ErrorList {
	ctx, logger := logging.GetLogComponent(ctx, "webhook_paasconfig_validateQuotaSettings")
	var allErrs field.ErrorList

	// Ensure DefQuota is set
	if qs.DefQuota == nil {
		logger.Error().Msg("capability is missing required DefQuota")
		allErrs = append(allErrs, field.Required(
			field.NewPath("ConfigCapability").Child("QuotaSettings").Child("DefQuota"),
			"field is required"))
	}

	// Ensure Ratio is sane
	if qs.Ratio < 0.0 || qs.Ratio > 1.0 {
		logger.Error().Msg("capability has an invalid Ratio, must be between 0.0 and 1.0")
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("ConfigCapability").Child("QuotaSettings").Child("Ratio"),
			qs.Ratio,
			"field is required"))
	}

	for resourceName, defQuantity := range qs.DefQuota {
		// Ensure DefQuota does not exceed MaxQuota
		if maxQuantity, exists := qs.MaxQuotas[resourceName]; exists {
			if defQuantity.Cmp(maxQuantity) > 0 {
				msg := fmt.Sprintf("capability has DefQuota %s exceeding MaxQuota %s for resource %s", defQuantity.String(), maxQuantity.String(), resourceName)
				logger.Error().Msg(msg)
				allErrs = append(allErrs, field.Invalid(
					field.NewPath("ConfigCapability").Child("QuotaSettings").Child("DefQuota"),
					qs.DefQuota,
					fmt.Sprintf("value of DefQuota exceeds MaxQuota for resource %s", resourceName)))
			}
		}

		// DefQuota should not be lower than MinQuota
		if minQuantity, exists := qs.MinQuotas[resourceName]; exists {
			if defQuantity.Cmp(minQuantity) < 0 {
				logger.Error().Msg(fmt.Sprintf("capability has DefQuota %s lower than MinQuota %s for resource %s", defQuantity.String(), minQuantity.String(), resourceName))
			}
		}
	}

	// Ensure MinQuota does not exceed MaxQuota
	for resourceName, minQuantity := range qs.MinQuotas {
		if maxQuantity, exists := qs.MaxQuotas[resourceName]; exists {
			if minQuantity.Cmp(maxQuantity) > 0 {
				logger.Error().Msg(fmt.Sprintf("capability has MinQuota %s exceeding MaxQuota %s for resource %s", minQuantity.String(), maxQuantity.String(), resourceName))
				allErrs = append(allErrs, field.Invalid(
					field.NewPath("ConfigCapability").Child("QuotaSettings").Child("MinQuotas"),
					minQuantity,
					fmt.Sprintf("value of MinQuota exceeds MaxQuota for resource %s", resourceName)))
			}
		}
	}

	// If Clusterwide is set to true, there should be no Min/Max quotas per namespace.
	if qs.Clusterwide {
		if len(qs.MinQuotas) > 0 || len(qs.MaxQuotas) > 0 {
			logger.Error().Msg("capability is marked as clusterwide but has MinQuotas or MaxQuotas defined, which is inconsistent")
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("ConfigCapability").Child("QuotaSettings").Child("ClusterWide"),
				qs.Clusterwide,
				"capability is marked as clusterwide but has MinQuotas or MaxQuotas defined, which is inconsistent"))
		}
	}

	// If DefQuota, MinQuotas, or MaxQuotas are provided, ensure they aren't empty maps.
	if qs.DefQuota != nil && len(qs.DefQuota) == 0 {
		msg := fmt.Errorf("capability has an empty DefQuota map, which is invalid")
		logger.Error().Msg(msg.Error())
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("ConfigCapability").Child("QuotaSettings").Child("DefQuota"),
			qs.DefQuota,
			msg.Error()))
	}
	if qs.MinQuotas != nil && len(qs.MinQuotas) == 0 {
		msg := fmt.Errorf("capability has an empty MinQuotas map, which is invalid")
		logger.Error().Msg(msg.Error())
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("ConfigCapability").Child("QuotaSettings").Child("MinQuotas"),
			qs.DefQuota,
			msg.Error()))
	}
	if qs.MaxQuotas != nil && len(qs.MaxQuotas) == 0 {
		msg := fmt.Errorf("capability has an empty MaxQuotas map, which is invalid")
		logger.Error().Msg(msg.Error())
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("ConfigCapability").Child("QuotaSettings").Child("MaxQuotas"),
			qs.DefQuota,
			msg.Error()))
	}

	// Every key in MinQuotas and MaxQuotas should exist in DefQuota to avoid inconsistencies.
	for resourceName := range qs.MinQuotas {
		if _, exists := qs.DefQuota[resourceName]; !exists {
			msg := fmt.Errorf("capability has MinQuota for resource %s that does not exist in DefQuota", resourceName)
			logger.Error().Msg(msg.Error())
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("ConfigCapability").Child("QuotaSettings").Child("MinQuotas"),
				resourceName,
				msg.Error()))
		}
	}
	for resourceName := range qs.MaxQuotas {
		if _, exists := qs.DefQuota[resourceName]; !exists {
			msg := fmt.Errorf("capability has MaxQuota for resource %s that does not exist in DefQuota", resourceName)
			logger.Error().Msg(msg.Error())
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("ConfigCapability").Child("QuotaSettings").Child("MaxQuotas"),
				resourceName,
				msg.Error()))
		}
	}

	return &allErrs
}

// func validateRoleMappings(ctx context.Context, config v1alpha1.PaasConfigSpec) *field.ErrorList     {}
// func validateDecryptKeyExists(ctx context.Context, config v1alpha1.PaasConfigSpec) *field.ErrorList {}

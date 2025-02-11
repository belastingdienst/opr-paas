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
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	if flderr := validatePaasConfigSpec(ctx, v.client, paasconfig.Spec); flderr != nil {
		allErrs = append(allErrs, flderr...)
	}

	// Ensure LDAP.Host is syntactically valid string, connection check is not done
	if valid, err := validate.Hostname(paasconfig.Spec.LDAP.Host); !valid {
		logger.Error().Msg(err.Error())
		path := field.NewPath("PaasConfig").Child("Spec").Child("LDAP")
		allErrs = append(allErrs, field.Invalid(path, paasconfig.Spec.LDAP.Host, err.Error()))
	}

	return warn, apierrors.NewInvalid(
		schema.GroupKind{Group: v1alpha1.GroupVersion.Group, Kind: "PaasConfig"},
		paasconfig.Name,
		allErrs,
	)

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

//----- actual checks

func validateNoPaasConfigExists(ctx context.Context, client client.Client) *field.Error {
	ctx, logger := logging.GetLogComponent(ctx, "webhook_paasconfig_validateNoPaasConfigExists")

	var list v1alpha1.PaasConfigList

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

func validatePaasConfigSpec(ctx context.Context, client client.Client, spec v1alpha1.PaasConfigSpec) field.ErrorList {
	ctx, logger := logging.GetLogComponent(ctx, "webhook_paasconfig_validatePaasConfig")
	var allErrs field.ErrorList
	childPath := field.NewPath("spec")

	allErrs = append(allErrs, validateDecryptKeysSecretExists(ctx, client, spec.DecryptKeysSecret, childPath)...)
	allErrs = append(allErrs, validateCapabilities(spec.Capabilities, childPath)...)

	if len(allErrs) > 0 {
		logger.Error().Strs("validation_errors", formatFieldErrors(allErrs)).Msg("encountered errors during validation of PaasConfig")
	}

	return allErrs
}

func validateCapabilities(capabilities v1alpha1.ConfigCapabilities, rootPath *field.Path) field.ErrorList {
	childPath := rootPath.Child("capabilities")

	var allErrs field.ErrorList

	for name, capability := range capabilities {
		allErrs = append(allErrs, validateCapability(name, capability, childPath)...)
	}

	return allErrs
}

func validateCapability(name string, cap v1alpha1.ConfigCapability, rootPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	childPath := rootPath.Key(name)

	allErrs = append(allErrs, validateQuotaSettings(cap.QuotaSettings, childPath)...)
	allErrs = append(allErrs, validateCustomFields(cap.CustomFields, childPath)...)

	return allErrs
}

func validateQuotaSettings(qs v1alpha1.ConfigQuotaSettings, rootPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	childPath := rootPath.Child("quotasettings")

	for resourceName, defQuantity := range qs.DefQuota {
		// Ensure DefQuota does not exceed MaxQuota
		if maxQuantity, exists := qs.MaxQuotas[resourceName]; exists {
			if defQuantity.Cmp(maxQuantity) > 0 {
				allErrs = append(allErrs, field.Invalid(
					childPath.Child("defquota").Key(string(resourceName)),
					qs.DefQuota,
					"value of DefQuota exceeds MaxQuota"))
			}
		}

		// DefQuota should not be lower than MinQuota
		if minQuantity, exists := qs.MinQuotas[resourceName]; exists {
			if defQuantity.Cmp(minQuantity) < 0 {
				allErrs = append(allErrs, field.Invalid(
					childPath.Child("defquota").Key(string(resourceName)),
					qs.DefQuota,
					"value of DefQuota is lower than MinQuota"))
			}
		}
	}

	for resourceName, minQuantity := range qs.MinQuotas {
		// Ensure MinQuota does not exceed MaxQuota
		if maxQuantity, exists := qs.MaxQuotas[resourceName]; exists {
			if minQuantity.Cmp(maxQuantity) > 0 {
				allErrs = append(allErrs, field.Invalid(
					childPath.Child("minquotas").Key(string(resourceName)),
					minQuantity,
					"value of MinQuota exceeds MaxQuota"))
			}
		}
	}

	for resourceName, maxQuantity := range qs.MaxQuotas {
		// Ensure MaxQuota is not less than MinQuota
		if minQuantity, exists := qs.MinQuotas[resourceName]; exists {
			if maxQuantity.Cmp(minQuantity) > 0 {
				allErrs = append(allErrs, field.Invalid(
					childPath.Child("maxquotas").Key(string(resourceName)),
					maxQuantity,
					"value of MaxQuota is less than MinQuota"))
			}
		}
	}

	// If Clusterwide is set to true, there should be no Min/Max quotas per namespace.
	if qs.Clusterwide {
		if len(qs.MinQuotas) > 0 || len(qs.MaxQuotas) > 0 {
			allErrs = append(allErrs, field.Invalid(
				childPath.Child("ClusterWide"),
				qs.Clusterwide,
				"marked as clusterwide but has MinQuotas / MaxQuotas defined"))
		}
	}

	// If DefQuota, MinQuotas, or MaxQuotas are provided, ensure they aren't empty maps.
	if qs.DefQuota != nil && len(qs.DefQuota) == 0 {
		allErrs = append(allErrs, field.Invalid(
			childPath.Child("defquota"),
			qs.DefQuota,
			"empty DefQuota map is invalid"))
	}
	if qs.MinQuotas != nil && len(qs.MinQuotas) == 0 {
		allErrs = append(allErrs, field.Invalid(
			childPath.Child("minquotas"),
			qs.MinQuotas,
			"empty MinQuotas map is invalid"))
	}
	if qs.MaxQuotas != nil && len(qs.MaxQuotas) == 0 {
		allErrs = append(allErrs, field.Invalid(
			childPath.Child("maxquotas"),
			qs.MaxQuotas,
			"empty MaxQuotas map is invalid"))
	}

	// Every key in MinQuotas and MaxQuotas should exist in DefQuota to avoid inconsistencies.
	for resourceName := range qs.MinQuotas {
		if _, exists := qs.DefQuota[resourceName]; !exists {
			allErrs = append(allErrs, field.Invalid(
				childPath.Child("minquotas").Key(resourceName.String()),
				qs.MinQuotas,
				"resource key does not exist in DefQuota"))
		}
	}
	for resourceName := range qs.MaxQuotas {
		if _, exists := qs.DefQuota[resourceName]; !exists {
			allErrs = append(allErrs, field.Invalid(
				childPath.Child("maxquotas").Key(resourceName.String()),
				qs.MaxQuotas,
				"resource key does not exist in DefQuota"))
		}
	}

	return allErrs
}

func validateCustomFields(customfields map[string]v1alpha1.ConfigCustomField, rootPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	childPath := rootPath.Child("customfields")

	for name, cf := range customfields {
		allErrs = append(allErrs, validateCustomField(name, cf, childPath)...)
	}

	return allErrs
}

func validateCustomField(name string, customfield v1alpha1.ConfigCustomField, rootPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	childPath := rootPath.Child("customfields").Key(name)

	// Can't set both Required and Default
	if customfield.Required && customfield.Default != "" {
		allErrs = append(allErrs, field.Invalid(
			childPath,
			"",
			"both Required and Default are set",
		))
	}

	if customfield.Validation != "" {
		// Must have compilable regex
		if valid, err := validate.StringIsRegex(customfield.Validation); !valid {
			allErrs = append(allErrs, field.Invalid(
				childPath.Child("validation"),
				name,
				err.Error()))
		}

		// Default field must conform to regex validation
		if customfield.Default != "" {
			if matched, err := regexp.Match(customfield.Validation, []byte(customfield.Default)); err != nil {
				allErrs = append(allErrs, field.InternalError(
					childPath.Child("default"),
					fmt.Errorf("error trying to validate using regex: %s", err.Error()),
				),
				)
			} else if !matched {
				allErrs = append(allErrs, field.Invalid(
					childPath.Child("default"),
					customfield.Default,
					fmt.Sprintf("value does not match %s", customfield.Validation)))
			}
		}
	}

	return allErrs
}

// validateDecryptKeysSecret ensures that the referenced Secret exists in the cluster.
func validateDecryptKeysSecretExists(ctx context.Context, k8sclient client.Client, secretRef v1alpha1.NamespacedName, rootPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	childPath := rootPath.Child("decryptkeyssecret")

	if secretRef.Name == "" || secretRef.Namespace == "" {
		allErrs = append(allErrs, field.Required(childPath, "DecryptKeysSecret is required and must have both name and namespace"))
		return allErrs
	}

	// Query the Kubernetes API to check if the Secret exists
	secret := &v1.Secret{}
	err := k8sclient.Get(ctx, client.ObjectKey{Namespace: secretRef.Namespace, Name: secretRef.Name}, secret)
	if err != nil {
		allErrs = append(allErrs, field.NotFound(
			childPath,
			fmt.Sprintf("%s/%s", secretRef.Namespace, secretRef.Name),
		))
	}

	return allErrs
}

// func validateRoleMappings(ctx context.Context, config v1alpha1.PaasConfigSpec) *field.ErrorList     {}
// func validateDecryptKeyExists(ctx context.Context, config v1alpha1.PaasConfigSpec) *field.ErrorList {}

// Convert field.ErrorList to a slice of strings for logging purposes
func formatFieldErrors(allErrs field.ErrorList) []string {
	var errs []string
	for _, err := range allErrs {
		errs = append(errs, err.Error())
	}
	return errs
}

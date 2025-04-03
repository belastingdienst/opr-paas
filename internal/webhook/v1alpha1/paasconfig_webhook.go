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
	"github.com/belastingdienst/opr-paas/internal/templating"
	"github.com/belastingdienst/opr-paas/internal/validate"
	k8sv1 "k8s.io/api/core/v1"
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

//revive:line-length-limit:disable

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// revive:disable-line
// +kubebuilder:webhook:path=/validate-cpet-belastingdienst-nl-v1alpha1-paasconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=cpet.belastingdienst.nl,resources=paasconfig,verbs=create;update,versions=v1alpha1,name=vpaasconfig-v1alpha1.kb.io,admissionReviewVersions=v1

//revive:line-length-limit:enable

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
	logger.Info().Msgf("validation for creation of PaasConfig %s", paasconfig.GetName())

	// Deny creation from secondary or more PaasConfig resources
	if warnings, flderr := validateNoPaasConfigExists(ctx, v.client); flderr != nil {
		warn = append(warn, warnings...)
		allErrs = append(allErrs, flderr...)
		return warn, apierrors.NewInvalid(
			schema.GroupKind{Group: v1alpha1.GroupVersion.Group, Kind: "PaasConfig"},
			paasconfig.Name,
			allErrs,
		)
	}

	// Ensure all required fields and values are there
	if warnings, flderr := validatePaasConfigSpec(ctx, v.client, paasconfig.Spec); flderr != nil || len(warnings) > 0 {
		warn = append(warn, warnings...)
		allErrs = append(allErrs, flderr...)
	}

	if len(allErrs) > 0 {
		return warn, apierrors.NewInvalid(
			schema.GroupKind{Group: v1alpha1.GroupVersion.Group, Kind: "PaasConfig"},
			paasconfig.Name,
			allErrs,
		)
	}

	return warn, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type PaasConfig.
func (v *PaasConfigCustomValidator) ValidateUpdate(
	ctx context.Context,
	oldObj, newObj runtime.Object,
) (warn admission.Warnings, err error) {
	var allErrs field.ErrorList
	paasconfig, ok := newObj.(*v1alpha1.PaasConfig)
	if !ok {
		return nil, fmt.Errorf("expected a PaasConfig object but got %T", newObj)
	}

	_, logger := logging.SetWebhookLogger(ctx, paasconfig)
	logger.Info().Msgf("validation for updating of PaasConfig %s", paasconfig.GetName())

	// Ensure all required fields and values are there
	if warnings, flderr := validatePaasConfigSpec(ctx, v.client, paasconfig.Spec); flderr != nil || len(warnings) > 0 {
		warn = append(warn, warnings...)
		allErrs = append(allErrs, flderr...)
	}

	// TODO(hikarukin): figure out what we need to check on update specifically
	logger.Debug().Msgf("old PaasConfig: %v", oldObj.(*v1alpha1.PaasConfig))

	if len(allErrs) > 0 {
		return warn, apierrors.NewInvalid(
			schema.GroupKind{Group: v1alpha1.GroupVersion.Group, Kind: "PaasConfig"},
			paasconfig.Name,
			allErrs,
		)
	}

	return warn, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type PaasConfig.
func (*PaasConfigCustomValidator) ValidateDelete(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	paasconfig, ok := obj.(*v1alpha1.PaasConfig)
	if !ok {
		return nil, fmt.Errorf("expected a PaasConfig object but got %T", obj)
	}

	_, logger := logging.SetWebhookLogger(ctx, paasconfig)
	logger.Info().Msgf("validation for deletion of PaasConfig %s", paasconfig.GetName())

	// Nothing to validate for deletion
	return nil, nil
}

// ----- actual checks

func validateNoPaasConfigExists(
	ctx context.Context,
	k8sClient client.Client,
) (warn admission.Warnings, allErrs field.ErrorList) {
	ctx, logger := logging.GetLogComponent(ctx, "webhook_paasconfig_validateNoPaasConfigExists")
	childPath := field.NewPath("spec")

	var list v1alpha1.PaasConfigList

	if err := k8sClient.List(ctx, &list); err != nil {
		err = fmt.Errorf("failed to retrieve PaasConfigList: %w", err)
		logger.Error().Msg(err.Error())
		allErrs = append(allErrs, field.InternalError(childPath, err))
		return nil, allErrs
	}

	if len(list.Items) > 0 {
		allErrs = append(allErrs, field.Forbidden(childPath, "another PaasConfig resource already exists"))
	}

	return nil, allErrs
}

func validatePaasConfigSpec(
	ctx context.Context,
	k8sClient client.Client,
	spec v1alpha1.PaasConfigSpec,
) (warn admission.Warnings, allErrs field.ErrorList) {
	ctx, logger := logging.GetLogComponent(ctx, "webhook_paasconfig_validatePaasConfig")
	childPath := field.NewPath("spec")

	// Ensure we generate some warnings if deprecated items are used
	if spec.ArgoPermissions.Header != "" {
		warn = append(warn, fmt.Sprintf("%s: %s", childPath.Child("argopermissions"), "deprecated"))
	}
	if spec.ExcludeAppSetName != "" {
		warn = append(warn, fmt.Sprintf("%s: %s", childPath.Child("excludeappsetname"), "deprecated"))
	}
	if spec.GroupSyncListKey != "" {
		warn = append(warn, fmt.Sprintf("%s: %s", childPath.Child("groupsynclistkey"), "deprecated"))
	}
	if spec.GroupSyncList.Name != "" {
		warn = append(warn, fmt.Sprintf("%s: %s", childPath.Child("groupsynclist"), "deprecated"))
	}

	// Ensure LDAP.Host is syntactically valid string, connection check is not done
	if spec.LDAP.Host != "" {
		if valid, err := validate.Hostname(spec.LDAP.Host); !valid {
			allErrs = append(allErrs, field.Invalid(
				childPath.Child("LDAP"),
				spec.LDAP.Host,
				err.Error(),
			))
		}
	}

	allErrs = append(allErrs, validateDecryptKeysSecretExists(ctx, k8sClient, spec.DecryptKeysSecret, childPath)...)
	allErrs = append(allErrs, validateValidationFields(spec.Validations, childPath)...)
	allErrs = append(allErrs, validateConfigCapabilityNames(spec, childPath)...)
	allErrs = append(allErrs, validateConfigCapabilities(spec.Capabilities, childPath)...)

	if len(allErrs) > 0 {
		logger.Error().Strs(
			"validation_errors",
			formatFieldErrors(allErrs),
		).Msg("encountered errors during validation of PaasConfig")
	}

	return warn, allErrs
}

func validateConfigCapabilities(capabilities v1alpha1.ConfigCapabilities, rootPath *field.Path) field.ErrorList {
	childPath := rootPath.Child("capabilities")

	var allErrs field.ErrorList

	for name, capability := range capabilities {
		allErrs = append(allErrs, validateConfigCapability(name, capability, childPath)...)
	}

	return allErrs
}

func validateConfigCapabilityNames(spec v1alpha1.PaasConfigSpec, rootPath *field.Path) field.ErrorList {
	var validationRE *regexp.Regexp
	childPath := rootPath.Child("capabilities")
	if spec.Validations == nil {
		return nil
	} else if validationRE = spec.Validations.GetValidationRE("paasConfig", "capabilityName"); validationRE == nil {
		return nil
	}

	var allErrs field.ErrorList

	for name := range spec.Capabilities {
		if !validationRE.Match([]byte(name)) {
			allErrs = append(allErrs, field.Invalid(
				childPath.Key(name),
				name,
				fmt.Sprintf("capability name does not match configured validation regex `%s`", validationRE.String()),
			))
		}
	}

	return allErrs
}

func validateConfigCapability(name string, capability v1alpha1.ConfigCapability, rootPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	childPath := rootPath.Key(name)

	allErrs = append(allErrs, validateConfigQuotaSettings(capability.QuotaSettings, childPath)...)
	allErrs = append(allErrs, validateConfigCustomFields(capability.CustomFields, childPath)...)

	return allErrs
}

func validateConfigQuotaSettings(qs v1alpha1.ConfigQuotaSettings, rootPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	childPath := rootPath.Child("quotasettings")

	allErrs = append(allErrs, validateConfigDefQuota(qs, childPath)...)
	allErrs = append(allErrs, validateConfigQuotaMinMax(qs, childPath)...)

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

	return allErrs
}

func validateConfigDefQuota(qs v1alpha1.ConfigQuotaSettings, childPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

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
		if minQuantity, exists := qs.MinQuotas[resourceName]; exists && !qs.Clusterwide {
			if defQuantity.Cmp(minQuantity) < 0 {
				allErrs = append(allErrs, field.Invalid(
					childPath.Child("defquota").Key(string(resourceName)),
					qs.DefQuota,
					"value of DefQuota is lower than MinQuota"))
			}
		}
	}

	return allErrs
}

func validateConfigQuotaMinMax(qs v1alpha1.ConfigQuotaSettings, childPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

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

		// Every key in MinQuotas should exist in DefQuota to avoid inconsistencies.
		if _, exists := qs.DefQuota[resourceName]; !exists {
			allErrs = append(allErrs, field.Invalid(
				childPath.Child("minquotas").Key(resourceName.String()),
				qs.MinQuotas,
				"resource key does not exist in DefQuota"))
		}
	}

	for resourceName, maxQuantity := range qs.MaxQuotas {
		// Ensure MaxQuota is not less than MinQuota
		if minQuantity, exists := qs.MinQuotas[resourceName]; exists {
			if maxQuantity.Cmp(minQuantity) < 0 {
				allErrs = append(allErrs, field.Invalid(
					childPath.Child("maxquotas").Key(string(resourceName)),
					maxQuantity,
					"value of MaxQuota is less than MinQuota"))
			}
		}

		// Every key in MaxQuotas should exist in DefQuota to avoid inconsistencies.
		if _, exists := qs.DefQuota[resourceName]; !exists {
			allErrs = append(allErrs, field.Invalid(
				childPath.Child("maxquotas").Key(resourceName.String()),
				qs.MaxQuotas,
				"resource key does not exist in DefQuota"))
		}
	}

	return allErrs
}

func validateConfigCustomFields(
	customfields map[string]v1alpha1.ConfigCustomField,
	rootPath *field.Path,
) field.ErrorList {
	var allErrs field.ErrorList
	childPath := rootPath.Child("customfields")

	for name, cf := range customfields {
		allErrs = append(allErrs, validateConfigCustomField(name, cf, childPath)...)
	}

	return allErrs
}

func validateConfigCustomField(
	name string,
	customfield v1alpha1.ConfigCustomField,
	rootPath *field.Path,
) field.ErrorList {
	var allErrs field.ErrorList
	childPath := rootPath.Child("customfields").Key(name)

	// Can't set a combination of Required, Default, and Template
	if (customfield.Required && customfield.Default != "") ||
		(customfield.Required && customfield.Template != "") ||
		(customfield.Template != "" && customfield.Default != "") {
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

	if customfield.Template != "" {
		err := templating.NewTemplater(v1alpha1.Paas{}, v1alpha1.PaasConfig{}).Verify(name, customfield.Template)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(
				childPath.Child("template"),
				customfield.Template,
				err.Error(),
			))
		}
	}

	return allErrs
}

// validateDecryptKeysSecret ensures that the referenced Secret exists in the cluster.
func validateDecryptKeysSecretExists(
	ctx context.Context,
	k8sclient client.Client,
	secretRef v1alpha1.NamespacedName,
	rootPath *field.Path,
) field.ErrorList {
	var allErrs field.ErrorList
	childPath := rootPath.Child("decryptkeyssecret")

	if secretRef.Name == "" || secretRef.Namespace == "" {
		allErrs = append(allErrs,
			field.Required(childPath, "DecryptKeysSecret is required and must have both name and namespace"))
		return allErrs
	}

	// Query the Kubernetes API to check if the Secret exists
	secret := &k8sv1.Secret{}
	err := k8sclient.Get(ctx, client.ObjectKey{Namespace: secretRef.Namespace, Name: secretRef.Name}, secret)
	if err != nil {
		allErrs = append(allErrs, field.NotFound(
			childPath,
			fmt.Sprintf("%s/%s", secretRef.Namespace, secretRef.Name),
		))
	}

	return allErrs
}

// validateDecryptKeysSecret ensures that the referenced Secret exists in the cluster.
func validateValidationFields(
	validations v1alpha1.PaasConfigValidations,
	rootPath *field.Path,
) field.ErrorList {
	var allErrs field.ErrorList
	childPath := rootPath.Child("validations")
	for resourceType, fieldValidations := range validations {
		for fieldName, validationRegexp := range fieldValidations {
			_, err := regexp.Compile(validationRegexp)
			if err != nil {
				allErrs = append(allErrs, field.Invalid(
					childPath,
					validationRegexp,
					fmt.Errorf(
						"failed to compile validation regexp for %s.%s: %w",
						resourceType,
						fieldName, err).Error(),
				))
			}
		}
	}
	return allErrs
}

// Convert field.ErrorList to a slice of strings for logging purposes
func formatFieldErrors(allErrs field.ErrorList) []string {
	var errs []string
	for _, err := range allErrs {
		errs = append(errs, err.Error())
	}
	return errs
}

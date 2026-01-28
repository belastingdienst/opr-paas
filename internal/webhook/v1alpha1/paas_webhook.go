/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

// Package v1alpha1 contains all webhook code for the v1alpha1 admission and conversion webhooks
package v1alpha1

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v4/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/v4/internal/config"
	"github.com/belastingdienst/opr-paas/v4/internal/logging"
	"github.com/belastingdienst/opr-paas/v4/pkg/quota"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// revive:disable:line-length-limit
const (
	orderedListWarning = "deprecation: list %s is not alphabetically sorted. When retrieving the list, the order may differ from the order in which it was created"
	disabledCapWarning = "deprecation: capability %s is disabled and will not be present when retrieving the Paas resource"
)

// SetupPaasWebhookWithManager registers the webhook for Paas in the manager.
func SetupPaasWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &v1alpha1.Paas{}).
		WithValidator(&PaasCustomValidator{client: mgr.GetClient()}).
		Complete()
}

// revive:disable:unused-parameter

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-cpet-belastingdienst-nl-v1alpha1-paas,mutating=false,failurePolicy=fail,sideEffects=None,groups=cpet.belastingdienst.nl,resources=paas,verbs=create;update,versions=v1alpha1,name=vpaas-v1alpha1.kb.io,admissionReviewVersions=v1

// PaasCustomValidator struct is responsible for validating the Paas resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
// +kubebuilder:object:generate=false

// revive:enable:line-length-limit

// PaasCustomValidator struct is responsible for validating the Paas resource when it is created, updated, or deleted.
type PaasCustomValidator struct {
	client client.Client
}

var _ admission.Validator[*v1alpha1.Paas] = &PaasCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateCreate(ctx context.Context, paas *v1alpha1.Paas) (admission.Warnings, error) {
	ctx, logger := logging.SetWebhookLogger(ctx, paas)

	myConf, err := config.GetConfigV1(ctx, v.client)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, config.ContextKeyPaasConfig, myConf)

	logger.Info().Msg("starting validation webhook for creation")

	return v.validate(ctx, paas)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateUpdate(
	ctx context.Context,
	_, nPaas *v1alpha1.Paas,
) (admission.Warnings, error) {
	ctx, logger := logging.SetWebhookLogger(ctx, nPaas)
	if nPaas.GetDeletionTimestamp() != nil {
		logger.Info().Msg("paas is being deleted")
		return nil, nil
	}
	logger.Info().Msg("starting validation webhook for update")

	myConf, err := config.GetConfigV1(ctx, v.client)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, config.ContextKeyPaasConfig, myConf)

	return v.validate(ctx, nPaas)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (*PaasCustomValidator) ValidateDelete(ctx context.Context, paas *v1alpha1.Paas) (admission.Warnings, error) {
	_, logger := logging.SetWebhookLogger(ctx, paas)
	logger.Info().Msg("starting validation webhook for deletion")

	// No validation needed for deletion.

	return nil, nil
}

type paasSpecValidator func(
	context.Context,
	client.Client,
	v1alpha1.PaasConfig,
	*v1alpha1.Paas,
) ([]*field.Error, error)

func (v *PaasCustomValidator) validate(ctx context.Context, paas *v1alpha1.Paas) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings []string
	if paas.DeletionTimestamp != nil {
		return nil, nil
	}
	conf, err := config.GetConfigFromContextV1(ctx)
	if err != nil {
		return nil, err
	}

	for _, val := range []paasSpecValidator{
		validatePaasName,
		validatePaasRequestor,
		validateCaps,
		validatePaasallowedQuotas,
		validateSecrets,
		validateCustomFields,
		validateGroupNames,
		validatePaasNamespaceNames,
	} {
		var fieldErrs []*field.Error
		if fieldErrs, err = val(ctx, v.client, conf, paas); err != nil {
			return nil, apierrors.NewInternalError(err)
		} else if fieldErrs != nil {
			allErrs = append(allErrs, fieldErrs...)
		}
	}

	warnings = append(warnings, validateGroups(paas.Spec.Groups)...)
	warnings = append(warnings, validateQuota(paas)...)
	warnings = append(warnings, validateExtraPerm(conf, paas)...)
	warnings = append(warnings, validateListSorted(paas.Spec.Namespaces, field.NewPath("spec").Child("namespaces"))...)
	warnings = append(warnings, validateDisabledCapabilities(paas.Spec.Capabilities)...)

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
func validateCaps(
	_ context.Context,
	_ client.Client,
	conf v1alpha1.PaasConfig,
	paas *v1alpha1.Paas,
) ([]*field.Error, error) {
	var errs []*field.Error

	for name := range paas.Spec.Capabilities {
		if _, ok := conf.Spec.Capabilities[name]; !ok {
			errs = append(errs, field.Invalid(
				field.NewPath("spec").Child("capabilities"),
				name,
				"capability not configured",
			))
		}
	}

	return errs, nil
}

// validatePaasName returns an error if the name of the paas does not meet validations.
func validatePaasName(
	_ context.Context,
	_ client.Client,
	conf v1alpha1.PaasConfig,
	paas *v1alpha1.Paas,
) ([]*field.Error, error) {
	var errs []*field.Error

	nameValidationRE := conf.GetValidationRE("paas", "name")
	if nameValidationRE == nil {
		return nil, nil
	}
	if !nameValidationRE.Match([]byte(paas.Name)) {
		errs = append(errs, field.Invalid(
			field.NewPath("metadata").Key("name"),
			paas.Name,
			fmt.Sprintf("paas name does not match configured validation regex `%s`", nameValidationRE.String()),
		))
	}

	return errs, nil
}

// validatePaasallowedQuotas returns errors if there are quota's with keys that do not meet validations
func validatePaasallowedQuotas(
	_ context.Context,
	_ client.Client,
	conf v1alpha1.PaasConfig,
	paas *v1alpha1.Paas,
) ([]*field.Error, error) {
	var errs []*field.Error
	nameValidationRE := conf.Spec.Validations.GetValidationRE("paas", "allowedQuotas")
	if nameValidationRE == nil {
		return nil, nil
	}

	quotas := map[*field.Path]quota.Quota{
		field.NewPath("spec", "quota"): paas.Spec.Quota,
	}
	cf := field.NewPath("spec", "capabilities")
	for name, c := range paas.Spec.Capabilities {
		quotas[cf.Key(name).Child("quota")] = c.Quota
	}

	for f, q := range quotas {
		for quotaKey := range q {
			if !nameValidationRE.Match([]byte(quotaKey)) {
				errs = append(errs, field.Invalid(
					f,
					quotaKey,
					fmt.Sprintf("quota is not allowed (allowed quotas: %s)", nameValidationRE.String()),
				))
			}
		}
	}
	return errs, nil
}

// RFC 1123 Label Names
// Some resource types require their names to follow the DNS label standard as defined in RFC 1123.
// This means the name must:
// - contain at most 63 characters
// - contain only lowercase alphanumeric characters or '-'
// - start with an alphabetic character
// - end with an alphanumeric character
var rfc1123LabelNamesRegex = regexp.MustCompile(`^[a-z]([-a-z0-9]{0,61}[a-z0-9])?$`)

// validatePaasNamespaceNames returns an error for every namespace that does not meet validations.
func validatePaasNamespaceNames(
	_ context.Context,
	_ client.Client,
	conf v1alpha1.PaasConfig,
	paas *v1alpha1.Paas,
) ([]*field.Error, error) {
	var errs []*field.Error

	// We use same value for paas.spec.namespaces and paasns.metadata.name validation.
	// Unless both are set.
	nameValidationRE := conf.GetValidationRE("paas", "namespaceName")
	if nameValidationRE == nil {
		nameValidationRE = conf.GetValidationRE("paasNs", "name")
	}

	for index, namespace := range paas.Spec.Namespaces {
		if !rfc1123LabelNamesRegex.MatchString(namespace) {
			errs = append(errs, field.Invalid(
				field.NewPath("spec").Child("namespaces").Index(index),
				namespace,
				"paas name does not match with RFC 1123 Label Names",
			))
		}
		if nameValidationRE != nil {
			if !nameValidationRE.Match([]byte(namespace)) {
				errs = append(errs, field.Invalid(
					field.NewPath("spec").Child("namespaces").Index(index),
					namespace,
					fmt.Sprintf("paas name does not match configured validation regex `%s`", nameValidationRE.String()),
				))
			}
		}
	}

	return errs, nil
}

// validatePaasRequestor returns an error if The requestor field in a Paas does not meet with validation RE
func validatePaasRequestor(
	_ context.Context,
	_ client.Client,
	conf v1alpha1.PaasConfig,
	paas *v1alpha1.Paas,
) ([]*field.Error, error) {
	var errs []*field.Error

	nameValidationRE := conf.GetValidationRE("paas", "requestor")
	if nameValidationRE == nil {
		return nil, nil
	}
	if !nameValidationRE.Match([]byte(paas.Spec.Requestor)) {
		errs = append(errs, field.Invalid(
			field.NewPath("spec").Key("requestor"),
			paas.Name,
			fmt.Sprintf("paas requestor does not match configured validation regex `%s`", nameValidationRE.String()),
		))
	}

	return errs, nil
}

// validateGroupNames returns an error for every group name that does not meet validations RE
func validateGroupNames(
	_ context.Context,
	_ client.Client,
	conf v1alpha1.PaasConfig,
	paas *v1alpha1.Paas,
) ([]*field.Error, error) {
	var errs []*field.Error
	groupNameValidationRE := conf.GetValidationRE("paas", "groupName")
	if groupNameValidationRE == nil {
		return nil, nil
	}

	for groupName := range paas.Spec.Groups {
		if !groupNameValidationRE.Match([]byte(groupName)) {
			errs = append(errs, field.Invalid(
				field.NewPath("spec").Child("groups").Key(groupName),
				groupName,
				fmt.Sprintf("group name does not match configured validation regex `%s`",
					groupNameValidationRE.String()),
			))
		}
	}

	return errs, nil
}

func validateSecrets(
	ctx context.Context,
	k8sClient client.Client,
	conf v1alpha1.PaasConfig,
	paas *v1alpha1.Paas,
) ([]*field.Error, error) {
	decryptRes := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      conf.Spec.DecryptKeysSecret.Name,
		Namespace: conf.Spec.DecryptKeysSecret.Namespace,
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
func validateCustomFields(
	_ context.Context,
	_ client.Client,
	conf v1alpha1.PaasConfig,
	paas *v1alpha1.Paas,
) ([]*field.Error, error) {
	var errs []*field.Error

	for cname, c := range paas.Spec.Capabilities {
		// validateCaps() has already ensured the capability configuration exists
		if _, err := c.CapExtraFields(conf.Spec.Capabilities[cname].CustomFields); err != nil {
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
func validateGroups(groups v1alpha1.PaasGroups) (warnings []string) {
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

// validateQuota returns a warning when higher limits are configured than requests for the Paas / capability quotas.
func validateQuota(paas *v1alpha1.Paas) (warnings []string) {
	quotas := map[*field.Path]quota.Quota{
		field.NewPath("spec", "quota"): paas.Spec.Quota,
	}
	cf := field.NewPath("spec", "capabilities")
	for name, c := range paas.Spec.Capabilities {
		quotas[cf.Key(name).Child("quota")] = c.Quota
	}

	for f, q := range quotas {
		reqc, reqcok := q[corev1.ResourceRequestsCPU]
		limc, limcok := q[corev1.ResourceLimitsCPU]

		if reqcok && limcok && reqc.Cmp(limc) > 0 {
			warnings = append(warnings,
				fmt.Sprintf("%s CPU resource request (%s) higher than limit (%s)", f, reqc.String(), limc.String()))
		}

		reqm, reqmok := q[corev1.ResourceRequestsMemory]
		limm, limmok := q[corev1.ResourceLimitsMemory]

		if reqmok && limmok && reqm.Cmp(limm) > 0 {
			warnings = append(warnings,
				fmt.Sprintf("%s memory resource request (%s) higher than limit (%s)", f, reqm.String(), limm.String()))
		}
	}

	return warnings
}

// validateExtraPerm returns a warning when extra permissions are requested for a capability that are not configured.
func validateExtraPerm(conf v1alpha1.PaasConfig, paas *v1alpha1.Paas) (warnings []string) {
	for cname, c := range paas.Spec.Capabilities {
		if c.ExtraPermissions && conf.Spec.Capabilities[cname].ExtraPermissions == nil {
			warnings = append(warnings, fmt.Sprintf(
				"%s capability does not have extra permissions configured",
				field.NewPath("spec", "capabilities").Key(cname).Child("extra_permissions"),
			))
		}
	}

	return warnings
}

// validateListSorted returns a warning when the list is not sorted in which case the get would return something else
// (without capability) then create
func validateListSorted(
	list []string,
	label *field.Path,
) (warnings []string) {
	if !sort.SliceIsSorted(list, func(i, j int) bool {
		return list[i] < list[j]
	}) {
		warnings = append(warnings, fmt.Sprintf(orderedListWarning, label))
	}

	return warnings
}

// validateDisabledCapabilities returns a warning when one or more capabilities have enabled set to false
// in which case the get would return something else (without capability) then create
func validateDisabledCapabilities(
	capabilities v1alpha1.PaasCapabilities,
) (warnings []string) {
	for capName, capConfig := range capabilities {
		if !capConfig.Enabled {
			warnings = append(warnings, fmt.Sprintf(disabledCapWarning, capName))
		}
	}
	return warnings
}

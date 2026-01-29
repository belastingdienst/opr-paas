/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v4/internal/config"
	"github.com/belastingdienst/opr-paas/v4/internal/logging"
	"github.com/belastingdienst/opr-paas/v4/internal/utils"
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

// SetupPaasWebhookWithManager registers the webhook for Paas in the manager.
func SetupPaasWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &v1alpha2.Paas{}).
		WithValidator(&PaasCustomValidator{client: mgr.GetClient()}).
		Complete()
}

// revive:disable:line-length-limit

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-cpet-belastingdienst-nl-v1alpha2-paas,mutating=false,failurePolicy=fail,sideEffects=None,groups=cpet.belastingdienst.nl,resources=paas,verbs=create;update,versions=v1alpha2,name=vpaas-v1alpha2.kb.io,admissionReviewVersions=v1

// revive:enable:line-length-limit

// PaasCustomValidator struct is responsible for validating the Paas resource when it is created, updated, or deleted.
type PaasCustomValidator struct {
	client client.Client
}

var _ admission.Validator[*v1alpha2.Paas] = &PaasCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateCreate(ctx context.Context, paas *v1alpha2.Paas) (admission.Warnings, error) {
	ctx, logger := logging.SetWebhookLogger(ctx, paas)
	myConfig, err := config.GetConfig(ctx, v.client)
	if err != nil {
		return nil, err
	}
	// Updates context to include paasConfig
	ctx = context.WithValue(ctx, config.ContextKeyPaasConfig, myConfig)
	logger.Info().Msg("starting validation webhook for creation")

	return v.validate(ctx, paas)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateUpdate(
	ctx context.Context,
	_, nPaas *v1alpha2.Paas,
) (admission.Warnings, error) {
	ctx, logger := logging.SetWebhookLogger(ctx, nPaas)
	if nPaas.GetDeletionTimestamp() != nil {
		logger.Info().Msg("paas is being deleted")
		return nil, nil
	}
	logger.Info().Msg("starting validation webhook for update")

	myConfig, err := config.GetConfig(ctx, v.client)
	if err != nil {
		return nil, err
	}
	// Updates context to include paasConfig
	ctx = context.WithValue(ctx, config.ContextKeyPaasConfig, myConfig)
	return v.validate(ctx, nPaas)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (*PaasCustomValidator) ValidateDelete(ctx context.Context, paas *v1alpha2.Paas) (admission.Warnings, error) {
	_, logger := logging.SetWebhookLogger(ctx, paas)
	logger.Info().Msg("starting validation webhook for deletion")

	// No validation needed for deletion.

	return nil, nil
}

type paasSpecValidator func(
	context.Context,
	client.Client,
	v1alpha2.PaasConfig,
	*v1alpha2.Paas,
) ([]*field.Error, error)

func (v *PaasCustomValidator) validate(ctx context.Context, paas *v1alpha2.Paas) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings []string
	ctx, logger := logging.GetLogComponent(ctx, logging.WebhookPaasComponentV2)
	conf, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return nil, err
	}
	// Check for uninitialized config
	if conf.Spec.DecryptKeysSecret.Name == "" {
		return nil, apierrors.NewInternalError(errors.New("uninitialized PaasConfig"))
	}

	for _, val := range []paasSpecValidator{
		validatePaasName,
		validatePaasRequestor,
		validateCaps,
		validatePaasallowedQuotas,
		validatePaasSecrets,
		validateCustomFields,
		validateGroupNames,
		validatePaasNamespaceNames,
		validatePaasNamespaceGroups,
		validateAppNamespaceQuota,
	} {
		if errs, validationErr := val(ctx, v.client, conf, paas); validationErr != nil {
			return nil, apierrors.NewInternalError(validationErr)
		} else if errs != nil {
			allErrs = append(allErrs, errs...)
		}
	}

	groupWarnings, groupErrors := v.validateGroups(paas.Spec.Groups, conf.Spec.FeatureFlags.GroupUserManagement)
	warnings = append(warnings, groupWarnings...)
	allErrs = append(allErrs, groupErrors...)
	warnings = append(warnings, v.validateQuota(paas)...)
	warnings = append(warnings, v.validateExtraPerm(conf, paas)...)

	if len(allErrs) == 0 && len(warnings) == 0 {
		logger.Info().Msg("validate ok")
		return nil, nil
	} else if len(allErrs) == 0 {
		return warnings, nil
	}

	return warnings, apierrors.NewInvalid(
		schema.GroupKind{Group: v1alpha2.GroupVersion.Group, Kind: "Paas"},
		paas.Name,
		allErrs,
	)
}

// validateCaps returns an error if any of the passed capabilities is not configured.
func validateCaps(
	ctx context.Context,
	k8sClient client.Client,
	conf v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
) ([]*field.Error, error) {
	var errs []*field.Error

	rsa, err := getCryptInstance(ctx, k8sClient, conf, paas.Name)
	if err != nil {
		return nil, err
	}

	for name, capability := range paas.Spec.Capabilities {
		if _, ok := conf.Spec.Capabilities[name]; !ok {
			errs = append(errs, field.Invalid(
				field.NewPath("spec").Child("capabilities"),
				name,
				"capability not configured",
			))
		} else {
			errs = append(errs, validateSecrets(
				capability.Secrets,
				rsa,
				field.NewPath("spec").Child("capabilities").Key(name).Child("secrets"),
			)...)
		}
	}

	return errs, nil
}

// validatePaasName returns an error if the name of the paas does not meet validations.
func validatePaasName(
	_ context.Context,
	_ client.Client,
	conf v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
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
	conf v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
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
	conf v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
) ([]*field.Error, error) {
	const validNsNameLength = 63
	var errs []*field.Error

	// We use same value for paas.spec.namespaces and paasns.metadata.name validation.
	// Unless both are set.
	nameValidationRE := conf.GetValidationRE("paas", "namespaceName")
	if nameValidationRE == nil {
		nameValidationRE = conf.GetValidationRE("paasNs", "name")
	}
	if nameValidationRE == nil {
		return nil, nil
	}
	for namespace := range paas.Spec.Namespaces {
		if !rfc1123LabelNamesRegex.MatchString(namespace) {
			errs = append(errs, field.Invalid(
				field.NewPath("spec").Child("namespaces").Key(namespace),
				namespace,
				"paas name does not match with RFC 1123 Label Names",
			))
		}
		nsName := utils.Join(paas.Name, namespace)
		if len(nsName) > validNsNameLength {
			errs = append(errs, field.Invalid(
				field.NewPath("metadata").Key("name"),
				nsName,
				"namespace name combined with paasns name too long",
			))
		}

		if !nameValidationRE.Match([]byte(namespace)) {
			errs = append(errs, field.Invalid(
				field.NewPath("spec").Child("namespaces"),
				namespace,
				fmt.Sprintf("paas name does not match configured validation regex `%s`", nameValidationRE.String()),
			))
		}
	}

	return errs, nil
}

// validatePaasNamespaceGroups ensures each group referenced in a namespace definition matches a group defined in the
// Paas.
func validatePaasNamespaceGroups(
	_ context.Context,
	_ client.Client,
	_ v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
) (ferrs []*field.Error, _ error) {
	for nsname, ns := range paas.Spec.Namespaces {
		for _, g := range ns.Groups {
			if _, ok := paas.Spec.Groups[g]; !ok {
				groups := slices.Collect(maps.Keys(paas.Spec.Groups))
				slices.Sort(groups)
				ferrs = append(ferrs, &field.Error{
					Type:     field.ErrorTypeInvalid,
					Field:    field.NewPath("spec").Child("namespaces").Key(nsname).Child("groups").String(),
					BadValue: g,
					Detail:   fmt.Errorf("does not exist in paas groups (%v)", strings.Join(groups, ", ")).Error(),
				})
			}
		}
	}

	return ferrs, nil
}

// validatePaasRequestor returns an error if The requestor field in a Paas does not meet with validation RE
func validatePaasRequestor(
	_ context.Context,
	_ client.Client,
	conf v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
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
	conf v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
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

func validatePaasSecrets(
	ctx context.Context,
	k8sClient client.Client,
	conf v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
) ([]*field.Error, error) {
	rsa, err := getCryptInstance(ctx, k8sClient, conf, paas.Name)
	if err != nil {
		return nil, err
	}

	return validateSecrets(paas.Spec.Secrets, rsa, field.NewPath("spec").Child("secrets")), nil
}

// validateCustomFields ensures that for a given capability in the Paas:
//   - all custom fields are configured for that capability in the PaasConfig
//   - all custom fields pass regular expression validation as configured in the PaasConfig if present
//
// Returns an internal error if the validation regexp cannot be compiled.
func validateCustomFields(
	_ context.Context,
	_ client.Client,
	conf v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
) ([]*field.Error, error) {
	var errs []*field.Error

	for cname, c := range paas.Spec.Capabilities {
		// validateCaps() has already ensured the capability configuration exists
		if err := c.ValidateCapExtraFields(conf.Spec.Capabilities[cname].CustomFields); err != nil {
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
func (*PaasCustomValidator) validateGroups(groups v1alpha2.PaasGroups,
	groupUserFeatureFlag string,
) (warnings []string, errs []*field.Error) {
	for key, grp := range groups {
		if len(grp.Query) > 0 && len(grp.Users) > 0 {
			warnings = append(warnings, fmt.Sprintf(
				"%s contains both users and query, the users will be ignored",
				field.NewPath("spec").Child("groups").Key(key),
			))
		}
		if len(grp.Users) > 0 {
			switch groupUserFeatureFlag {
			case "warn":
				warnings = append(warnings, fmt.Sprintf(
					"group %s has users which is discouraged",
					field.NewPath("spec").Child("groups").Key(key).Child("users"),
				))
			case "block":
				errs = append(errs, field.Invalid(
					field.NewPath("spec").Child("groups").Key(key).Child("users"),
					grp.Users,
					"groups with users is a disabled feature",
				))
			}
		}
	}

	return warnings, errs
}

// validateQuota returns a warning when higher limits are configured than requests for the Paas / capability quotas.
func (v *PaasCustomValidator) validateQuota(paas *v1alpha2.Paas) (warnings []string) {
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
func (v *PaasCustomValidator) validateExtraPerm(conf v1alpha2.PaasConfig, paas *v1alpha2.Paas) (warnings []string) {
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

// getCryptInstance returns a crypt based on the provided config and paasName
func getCryptInstance(
	ctx context.Context,
	k8sClient client.Client,
	conf v1alpha2.PaasConfig,
	paasName string,
) (*crypt.Crypt, error) {
	decryptRes := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      conf.Spec.DecryptKeysSecret.Name,
		Namespace: conf.Spec.DecryptKeysSecret.Namespace,
	}, decryptRes); err != nil {
		return nil, fmt.Errorf("could not retrieve decryption secret: %w", err)
	}

	keys, err := crypt.NewPrivateKeysFromSecretData(decryptRes.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private keys: %w", err)
	}

	rsa, err := crypt.NewCryptFromKeys(keys, "", paasName)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypt instance: %w", err)
	}

	return rsa, nil
}

// validateSecrets validates a map of Secrets based on a provided rsa
func validateSecrets(
	secrets map[string]string,
	rsa *crypt.Crypt,
	basePath *field.Path,
) []*field.Error {
	var errs []*field.Error
	for name, secret := range secrets {
		if _, err := rsa.Decrypt(secret); err != nil {
			errs = append(errs, field.Invalid(
				basePath.Key(name),
				secret,
				fmt.Sprintf("cannot be decrypted: %s", err),
			))
		}
	}
	return errs
}

// validate quota of namespaces that are not linked to a capability
func validateAppNamespaceQuota(
	ctx context.Context,
	k8sClient client.Client,
	_ v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
) ([]*field.Error, error) {
	var errs []*field.Error

	if len(paas.Spec.Quota) > 0 {
		return nil, nil
	}

	if len(paas.Spec.Namespaces) > 0 {
		errs = append(errs, field.Invalid(
			field.NewPath("spec", "namespaces"),
			fmt.Sprintf("%d", len(paas.Spec.Namespaces)),
			fmt.Sprintf("quota can not be empty when paas has namespaces (number of namespaces: %d)",
				len(paas.Spec.Namespaces)),
		))
	}

	for capName := range paas.Spec.Capabilities {
		capNS := strings.Join([]string{paas.Name, capName}, "-")

		pnsList := &v1alpha2.PaasNSList{}
		if err := k8sClient.List(ctx, pnsList, &client.ListOptions{Namespace: capNS}); err != nil {
			return errs, err
		}
		if len(pnsList.Items) > 0 {
			errs = append(errs, field.Invalid(
				field.NewPath("spec", "capabilities").Key(capName),
				fmt.Sprintf("%d", len(pnsList.Items)),
				"quota can not be empty when paas capability namespace has paasNs",
			))
		}
	}

	return errs, nil
}

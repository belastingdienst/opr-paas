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
	"slices"
	"strings"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"
	"github.com/belastingdienst/opr-paas/internal/quota"
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
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha2.Paas{}).
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

var _ webhook.CustomValidator = &PaasCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	paas, ok := obj.(*v1alpha2.Paas)
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
	oldObj, newObj runtime.Object,
) (admission.Warnings, error) {
	paas, ok := newObj.(*v1alpha2.Paas)
	if !ok {
		return nil, fmt.Errorf("expected a Paas object for the newObj but got %T", newObj)
	}
	ctx, logger := logging.SetWebhookLogger(ctx, paas)
	if paas.GetDeletionTimestamp() != nil {
		logger.Info().Msg("paas is being deleted")
		return nil, nil
	}
	logger.Info().Msg("starting validation webhook for update")

	return v.validate(ctx, paas)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (*PaasCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	paas, ok := obj.(*v1alpha2.Paas)
	if !ok {
		return nil, fmt.Errorf("expected a Paas object but got %T", obj)
	}
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
	ctx, logger := logging.GetLogComponent(ctx, "webhook_paas_validate")
	conf := config.GetConfig()
	// Check for uninitialized config
	if conf.Spec.DecryptKeysSecret.Name == "" {
		return nil, apierrors.NewInternalError(errors.New("uninitialized PaasConfig"))
	}

	for _, val := range []paasSpecValidator{
		validatePaasName,
		validatePaasRequestor,
		validateCaps,
		validatePaasSecrets,
		validateCustomFields,
		validateGroupNames,
		validatePaasNamespaceNames,
		validatePaasNamespaceGroups,
	} {
		if errs, err := val(ctx, v.client, conf, paas); err != nil {
			return nil, apierrors.NewInternalError(err)
		} else if errs != nil {
			allErrs = append(allErrs, errs...)
		}
	}

	warnings = append(warnings, v.validateGroups(paas.Spec.Groups)...)
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
	_ client.Client,
	conf v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
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
	ctx context.Context,
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

// validatePaasNamespaceNames returns an error for every namespace that does not meet validations.
func validatePaasNamespaceNames(
	ctx context.Context,
	_ client.Client,
	conf v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
) ([]*field.Error, error) {
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
	conf v1alpha2.PaasConfig,
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
	ctx context.Context,
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
	ctx context.Context,
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
	decryptRes := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      conf.Spec.DecryptKeysSecret.Name,
		Namespace: conf.Spec.DecryptKeysSecret.Namespace,
	}, decryptRes); err != nil {
		return nil, fmt.Errorf("could not retrieve decryption secret: %w", err)
	}

	keys, _ := crypt.NewPrivateKeysFromSecretData(decryptRes.Data)
	rsa, _ := crypt.NewCryptFromKeys(keys, "", paas.Name)

	var errs []*field.Error
	for name, secret := range paas.Spec.Secrets {
		if _, err := rsa.Decrypt(secret); err != nil {
			errs = append(errs, field.Invalid(
				field.NewPath("spec").Child("secrets"),
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
	ctx context.Context,
	_ client.Client,
	conf v1alpha2.PaasConfig,
	paas *v1alpha2.Paas,
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
func (*PaasCustomValidator) validateGroups(groups v1alpha2.PaasGroups) (warnings []string) {
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

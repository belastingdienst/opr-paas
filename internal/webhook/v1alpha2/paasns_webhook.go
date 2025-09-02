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
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/config"
	"github.com/belastingdienst/opr-paas/v3/internal/logging"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupPaasNsWebhookWithManager registers the webhook for PaasNs in the manager.
func SetupPaasNsWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1alpha2.PaasNS{}).
		WithValidator(&PaasNSCustomValidator{client: mgr.GetClient()}).
		Complete()
}

//revive:disable:line-length-limit

// +kubebuilder:webhook:path=/validate-cpet-belastingdienst-nl-v1alpha2-paasns,mutating=false,failurePolicy=fail,sideEffects=None,groups=cpet.belastingdienst.nl,resources=paasns,verbs=create;update,versions=v1alpha2,name=vpaasns-v1alpha2.kb.io,admissionReviewVersions=v1

// PaasNSCustomValidator struct is responsible for validating the PaasNS resource when it is created, updated, or deleted.
// +kubebuilder:object:generate=false
type PaasNSCustomValidator struct {
	client client.Client
}

//revive:enable:line-length-limit

type paasNsSpecValidator func(
	context.Context,
	client.Client,
	v1alpha2.PaasConfig,
	v1alpha2.Paas,
	v1alpha2.PaasNS,
) ([]*field.Error, error)

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type PaasNS.
func (v *PaasNSCustomValidator) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (w admission.Warnings, err error) {
	var errs field.ErrorList
	paasns, ok := obj.(*v1alpha2.PaasNS)
	ctx, _ = logging.SetWebhookLogger(ctx, paasns)
	ctx, logger := logging.GetLogComponent(ctx, logging.WebhookPaasNSComponentV2)
	logger.Info().Msgf("starting validation webhook for create")

	if !ok {
		return nil, &field.Error{
			Type:   field.ErrorTypeTypeInvalid,
			Detail: fmt.Errorf("expected a PaasNS object but got %T", obj).Error(),
		}
	}

	paas, err := paasNStoPaas(ctx, v.client, paasns)
	if err != nil {
		// code to Err when the referenced Paas does not exist
		errs = append(errs, &field.Error{
			Type:     field.ErrorTypeTypeInvalid,
			Field:    field.NewPath("spec").Child("paas").String(),
			BadValue: paasns.Spec.Paas,
			Detail:   err.Error(),
		})
		return w, errs.ToAggregate()
	}

	myConfig, err := config.GetConfigWithError()
	if err != nil {
		errs = append(errs, field.InternalError(
			field.NewPath("paasconfig"),
			fmt.Errorf("unable to retrieve paasconfig: %s", err),
		))
	}

	for _, validator := range []paasNsSpecValidator{
		validatePaasNsName,
		validatePaasNsGroups,
		validatePaasNsSecrets,
	} {
		var fieldErrs []*field.Error
		fieldErrs, err = validator(ctx, v.client, *myConfig, *paas, *paasns)
		if err != nil {
			return nil, err
		}
		errs = append(errs, fieldErrs...)
	}

	if len(errs) == 0 {
		return nil, nil
	}

	return w, errs.ToAggregate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type PaasNS.
func (v *PaasNSCustomValidator) ValidateUpdate(
	ctx context.Context,
	oldObj,
	newObj runtime.Object,
) (w admission.Warnings, err error) {
	var errs field.ErrorList
	oldPaasns, ok := oldObj.(*v1alpha2.PaasNS)
	if !ok {
		return nil, &field.Error{
			Type:   field.ErrorTypeTypeInvalid,
			Detail: fmt.Errorf("expected a PaasNS object but got %T", oldObj).Error(),
		}
	}

	ctx, _ = logging.SetWebhookLogger(ctx, oldPaasns)
	ctx, logger := logging.GetLogComponent(ctx, logging.WebhookPaasNSComponentV2)

	newPaasns, ok := newObj.(*v1alpha2.PaasNS)
	if !ok {
		return nil, &field.Error{
			Type:   field.ErrorTypeTypeInvalid,
			Detail: fmt.Errorf("expected a PaasNS object but got %T", newObj).Error(),
		}
	}

	if newPaasns.GetDeletionTimestamp() != nil {
		logger.Info().Msg("paasns is being deleted")
		return nil, nil
	}
	logger.Info().Msg("starting validation webhook for update")

	// This will not occur.
	paas, _ := paasNStoPaas(ctx, v.client, newPaasns)

	for _, validator := range []paasNsSpecValidator{
		validatePaasNsGroups,
		validatePaasNsSecrets,
	} {
		myConfig := config.GetConfig()
		var fieldErrs []*field.Error
		fieldErrs, err = validator(ctx, v.client, myConfig, *paas, *newPaasns)
		if err != nil {
			return nil, err
		}
		errs = append(errs, fieldErrs...)
	}

	if len(errs) > 0 {
		return w, errs.ToAggregate()
	}
	return w, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type PaasNS.
func (*PaasNSCustomValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// validatePaasNsName returns an error when the naam of the PaasNs does not meet validations RE
func validatePaasNsName(
	_ context.Context,
	_ client.Client,
	conf v1alpha2.PaasConfig,
	_ v1alpha2.Paas,
	paasns v1alpha2.PaasNS,
) ([]*field.Error, error) {
	var fieldErrors []*field.Error
	if strings.Contains(paasns.Name, ".") {
		fieldErrors = append(fieldErrors, field.Invalid(
			field.NewPath("metadata").Key("name"),
			paasns.Name,
			"paasns name should not contain dots",
		))
	}
	nameValidationRE := conf.GetValidationRE("paasNs", "name")
	if nameValidationRE == nil {
		nameValidationRE = conf.GetValidationRE("paas", "namespaceName")
	}
	if nameValidationRE == nil {
		return fieldErrors, nil
	}
	if !nameValidationRE.Match([]byte(paasns.Name)) {
		fieldErrors = append(fieldErrors, field.Invalid(
			field.NewPath("metadata").Key("name"),
			paasns.Name,
			fmt.Sprintf("paasns name does not match configured validation regex `%s`", nameValidationRE.String()),
		))
	}
	return fieldErrors, nil
}

func validatePaasNsGroups(
	_ context.Context,
	_ client.Client,
	_ v1alpha2.PaasConfig,
	paas v1alpha2.Paas,
	paasns v1alpha2.PaasNS,
) ([]*field.Error, error) {
	var errs []*field.Error
	superGroups := maps.Keys(paas.Spec.Groups)
	uqSuperGroups := map[string]bool{}
	for group := range superGroups {
		uqSuperGroups[group] = true
	}
	// Err when the optional groupKey(s) don't exist in referenced Paas
	for _, group := range paasns.Spec.Groups {
		if _, exists := uqSuperGroups[group]; !exists {
			errs = append(errs, &field.Error{
				Type:     field.ErrorTypeInvalid,
				Field:    field.NewPath("spec").Child("groups").Key(group).String(),
				BadValue: group,
				Detail:   fmt.Errorf("group %s does not exist in paas groups (%v)", group, superGroups).Error(),
			})
		}
	}
	return errs, nil
}

func validatePaasNsSecrets(
	ctx context.Context,
	k8sClient client.Client,
	conf v1alpha2.PaasConfig,
	paas v1alpha2.Paas,
	paasns v1alpha2.PaasNS,
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
	for name, secret := range paasns.Spec.Secrets {
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

func paasNStoPaas(ctx context.Context, c client.Client, paasns *v1alpha2.PaasNS) (paas *v1alpha2.Paas, err error) {
	var ns corev1.Namespace
	ctx, logger := logging.GetLogComponent(ctx, logging.WebhookPaasNSComponentV2)
	if err = c.Get(
		context.Background(),
		types.NamespacedName{Name: paasns.GetNamespace()},
		&ns,
	); err != nil {
		logger.Error().Msgf("unable to get namespace where paasns resides: %s", err.Error())
		return nil, err
	}
	var paasNames []string
	for _, ref := range ns.OwnerReferences {
		if ref.Kind == "Paas" && ref.Controller != nil && *ref.Controller {
			paasNames = append(paasNames, ref.Name)
		}
	}
	if len(paasNames) == 0 {
		return nil, errors.New(
			"failed to get owner reference with kind paas and controller=true from namespace resource")
	} else if len(paasNames) > 1 {
		return nil, fmt.Errorf("found %d owner references with kind paas and controller=true", len(paasNames))
	}
	paasName := paasNames[0]
	if ns.Name != paasName && !strings.HasPrefix(ns.Name, paasName+"-") {
		return nil, fmt.Errorf(
			"namespace %s is not named after paas, and not prefixed with '%s-' (paasName from owner reference)",
			ns.Name, paasName)
	}
	paas = &v1alpha2.Paas{}
	err = c.Get(ctx, client.ObjectKey{
		Name: paasName,
	}, paas)
	if err != nil {
		err = fmt.Errorf("failed to get paas: %w", err)
		logger.Error().Msg(err.Error())
		return nil, err
	}
	return paas, nil
}

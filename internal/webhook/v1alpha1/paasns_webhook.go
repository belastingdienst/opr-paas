/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v3/api/v1alpha1"
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
	return ctrl.NewWebhookManagedBy(mgr).For(&v1alpha1.PaasNS{}).
		WithValidator(&PaasNSCustomValidator{client: mgr.GetClient()}).
		Complete()
}

//revive:disable:line-length-limit

// +kubebuilder:webhook:path=/validate-cpet-belastingdienst-nl-v1alpha1-paasns,mutating=false,failurePolicy=fail,sideEffects=None,groups=cpet.belastingdienst.nl,resources=paasns,verbs=create;update,versions=v1alpha1,name=vpaasns-v1alpha1.kb.io,admissionReviewVersions=v1

// PaasNSCustomValidator struct is responsible for validating the PaasNS resource when it is created, updated, or deleted.
// +kubebuilder:object:generate=false
type PaasNSCustomValidator struct {
	client client.Client
}

//revive:enable:line-length-limit

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type PaasNS.
func (v *PaasNSCustomValidator) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (w admission.Warnings, err error) {
	var errs field.ErrorList
	paasns, ok := obj.(*v1alpha1.PaasNS)
	ctx, _ = logging.SetWebhookLogger(ctx, paasns)
	ctx, logger := logging.GetLogComponent(ctx, logging.WebhookPaasNSComponentV1)
	logger.Info().Msgf("starting validation webhook for create")

	myConf, err := config.GetConfigV1(ctx, v.client)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, contextKeyPaasConfig, myConf)

	if !ok {
		return nil, &field.Error{
			Type:   field.ErrorTypeTypeInvalid,
			Detail: fmt.Errorf("expected a PaasNS object but got %T", obj).Error(),
		}
	}

	errs = append(errs, v.validatePaasNsName(ctx, paasns.Name)...)

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

	// Err when a sshSecret can't be decrypted
	errs = append(errs, compareGroups(paasns.Spec.Groups, paas.Spec.Groups.Keys())...)

	// Code to Err when an sshSecret can't be decrypted
	var validated validatedSecrets
	validated.appendFromPaas(*paas)
	getRsaFunc := func() (cr *crypt.Crypt, err error) {
		return getRsa(ctx, v.client, paasns.Spec.Paas)
	}
	errs = append(errs, validated.compareSecrets(paasns.Spec.SSHSecrets, getRsaFunc)...)

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
	oldPaasns, ok := oldObj.(*v1alpha1.PaasNS)
	if !ok {
		return nil, &field.Error{
			Type:   field.ErrorTypeTypeInvalid,
			Detail: fmt.Errorf("expected a PaasNS object but got %T", oldObj).Error(),
		}
	}

	ctx, _ = logging.SetWebhookLogger(ctx, oldPaasns)
	ctx, logger := logging.GetLogComponent(ctx, logging.WebhookPaasNSComponentV1)

	newPaasns, ok := newObj.(*v1alpha1.PaasNS)
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

	if oldPaasns.Spec.Paas != newPaasns.Spec.Paas {
		return w, &field.Error{
			Type:     field.ErrorTypeNotSupported,
			Field:    field.NewPath("spec").Child("paas").String(),
			BadValue: newPaasns.Spec.Paas,
			Detail:   "field is immutable",
		}
	}

	paas, _ := paasNStoPaas(ctx, v.client, newPaasns)

	if !reflect.DeepEqual(oldPaasns.Spec.Groups, newPaasns.Spec.Groups) {
		// Raise errors when groups are not in paas
		errs = append(errs, compareGroups(newPaasns.Spec.Groups, paas.Spec.Groups.Keys())...)
	}

	// Err when an sshSecret can't be decrypted
	if !reflect.DeepEqual(oldPaasns.Spec.SSHSecrets, newPaasns.Spec.SSHSecrets) {
		var validated validatedSecrets
		// We don't have to validate what is in the Paas (already validated by Paas webhook)
		validated.appendFromPaas(*paas)
		// We don't have to validate what is in the previous PaasNs definition (already validated before)
		validated.appendFromPaasNS(*oldPaasns)
		getRsaFunc := func() (cr *crypt.Crypt, err error) {
			return getRsa(ctx, v.client, newPaasns.Spec.Paas)
		}
		errs = append(errs, validated.compareSecrets(newPaasns.Spec.SSHSecrets, getRsaFunc)...)
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

func paasNStoPaas(ctx context.Context, c client.Client, paasns *v1alpha1.PaasNS) (paas *v1alpha1.Paas, err error) {
	var ns corev1.Namespace
	ctx, logger := logging.GetLogComponent(ctx, logging.WebhookPaasNSComponentV1)
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
	paas = &v1alpha1.Paas{}
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

// compareGroups is a helper function to compare the list of groups in a PaasNS against the list in the Paas
func compareGroups(subGroups []string, superGroups []string) (errs field.ErrorList) {
	uqSuperGroups := map[string]bool{}
	for _, group := range superGroups {
		uqSuperGroups[group] = true
	}
	// Err when the optional groupKey(s) don't exist in referenced Paas
	for _, group := range subGroups {
		if _, exists := uqSuperGroups[group]; !exists {
			errs = append(errs, &field.Error{
				Type:     field.ErrorTypeInvalid,
				Field:    field.NewPath("spec").Child("groups").Key(group).String(),
				BadValue: group,
				Detail:   fmt.Errorf("group %s does not exist in paas groups (%v)", group, superGroups).Error(),
			})
		}
	}
	return errs
}

// validatePaasNsName returns an error when the naam of the PaasNs does not meet validations RE
func (v *PaasNSCustomValidator) validatePaasNsName(ctx context.Context, name string) (errs field.ErrorList) {
	conf, err := getConfigFromContext(ctx)
	if err != nil {
		errs = append(errs, field.InternalError(
			field.NewPath("paasconfig"),
			fmt.Errorf("unable to retrieve paasconfig: %s", err),
		))
		return errs
	}

	if strings.Contains(name, ".") {
		errs = append(errs, field.Invalid(
			field.NewPath("metadata").Key("name"),
			name,
			"paasns name should not contain dots",
		))
	}
	nameValidationRE := conf.GetValidationRE("paasNs", "name")
	if nameValidationRE == nil {
		nameValidationRE = conf.GetValidationRE("paas", "namespaceName")
	}
	if nameValidationRE == nil {
		return errs
	}
	if !nameValidationRE.Match([]byte(name)) {
		errs = append(errs, field.Invalid(
			field.NewPath("metadata").Key("name"),
			name,
			fmt.Sprintf("paasns name does not match configured validation regex `%s`", nameValidationRE.String()),
		))
	}
	return errs
}

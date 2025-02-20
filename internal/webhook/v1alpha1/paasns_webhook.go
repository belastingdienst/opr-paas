/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"
	"fmt"
	"reflect"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/belastingdienst/opr-paas/internal/logging"
	"k8s.io/apimachinery/pkg/runtime"
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

// +kubebuilder:webhook:path=/validate-cpet-belastingdienst-nl-v1alpha1-paasns,mutating=false,failurePolicy=fail,sideEffects=None,groups=cpet.belastingdienst.nl,resources=paasns,verbs=create;update,versions=v1alpha1,name=vpaasns-v1alpha1.kb.io,admissionReviewVersions=v1

// PaasNSCustomValidator struct is responsible for validating the PaasNS resource when it is created, updated, or deleted.
// +kubebuilder:object:generate=false
type PaasNSCustomValidator struct {
	client client.Client
}

// TODO devotional-phoenix-97: combine with controller code

// nssFromNs gets all PaasNs objects from a namespace and returns a list of all the corresponding namespaces
// It also returns PaasNS in those namespaces recursively.
func nssFromNs(ctx context.Context, c client.Client, ns string) (map[string]int, error) {
	nss := make(map[string]int)
	pnsList := &v1alpha1.PaasNSList{}
	if err := c.List(ctx, pnsList, &client.ListOptions{Namespace: ns}); err != nil {
		return nil, err
	}
	for _, pns := range pnsList.Items {
		nsName := pns.NamespaceName()
		if value, exists := nss[nsName]; exists {
			nss[nsName] = value + 1
		} else {
			nss[nsName] = 1
		}
		// Call myself (recursively)
		subNss, err := nssFromNs(ctx, c, nsName)
		if err != nil {
			return nil, err
		}
		for key, value := range subNss {
			nss[key] += value
		}
	}
	return nss, nil
}

// nssFromPaas accepts a Paas and returns a list of all namespaces managed by this Paas
// nssFromPaas uses nssFromNs which is recursive.
func nssFromPaas(ctx context.Context, c client.Client, paas *v1alpha1.Paas) (map[string]int, error) {
	finalNss := make(map[string]int)
	finalNss[paas.Name] = 1
	nss, err := nssFromNs(ctx, c, paas.Name)
	if err != nil {
		return nil, err
	}
	for key, value := range nss {
		finalNss[key] += value
	}
	return finalNss, nil
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type PaasNS.
func (v *PaasNSCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (w admission.Warnings, err error) {
	var errs field.ErrorList
	paasns, ok := obj.(*v1alpha1.PaasNS)
	ctx, _ = logging.SetWebhookLogger(ctx, paasns)
	ctx, logger := logging.GetLogComponent(ctx, "paasns_validate_create")
	logger.Info().Msgf("starting validation webhook for create")

	if !ok {
		return nil, &field.Error{
			Type:   field.ErrorTypeTypeInvalid,
			Detail: fmt.Errorf("expected a PaasNS object but got %T", obj).Error(),
		}
	}

	paas, err := getPaas(ctx, v.client, paasns.Spec.Paas)
	if err != nil {
		// code to Err when the referenced Paas does not exist
		errs = append(errs, &field.Error{
			Type:     field.ErrorTypeTypeInvalid,
			Field:    field.NewPath("spec").Child("paas").String(),
			BadValue: paasns.Spec.Paas,
			Detail:   fmt.Errorf("paas %s does not exist: %w", paasns.Spec.Paas, err).Error(),
		})
		return w, errs.ToAggregate()
	}

	if nss, err := nssFromPaas(ctx, v.client, paas); err != nil {
		errs = append(errs, &field.Error{
			Type:     field.ErrorTypeInvalid,
			Field:    field.NewPath("spec").Child("paas").String(),
			BadValue: paasns.Spec.Paas,
			Detail:   fmt.Errorf("cannot get nss for this paas: %w", err).Error(),
		})
	} else if _, exists := nss[paasns.Namespace]; !exists {
		errs = append(errs, &field.Error{
			Type:     field.ErrorTypeInvalid,
			Field:    field.NewPath("spec").Child("paas").String(),
			BadValue: paasns.Spec.Paas,
			Detail:   fmt.Errorf("paasns not in namespace belonging to paas %s", paas.Name).Error(),
		})
	}

	// Err when a sshSecret can't be decrypted
	errs = append(errs, compareGroups(paasns.Spec.Groups, paas.Spec.Groups.Keys())...)

	// Code to Err when an sshSecret can't be decrypted
	var validated validatedSecrets
	validated.appendFromPaas(*paas)
	getRsaFunc := func() (cr *crypt.Crypt, err error) {
		return getRsa(ctx, v.client, paasns.Spec.Paas)
	}
	errs = append(errs, validated.compareSecrets(paasns.Spec.SshSecrets, getRsaFunc)...)

	if len(errs) == 0 {
		return nil, nil
	}

	return w, errs.ToAggregate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type PaasNS.
func (v *PaasNSCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (w admission.Warnings, err error) {
	var errs field.ErrorList
	oldPaasns, ok := oldObj.(*v1alpha1.PaasNS)
	if !ok {
		return nil, &field.Error{
			Type:   field.ErrorTypeTypeInvalid,
			Detail: fmt.Errorf("expected a PaasNS object but got %T", oldObj).Error(),
		}
	}

	ctx, _ = logging.SetWebhookLogger(ctx, oldPaasns)
	ctx, logger := logging.GetLogComponent(ctx, "paasns_validate_update")

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

	paas, _ := getPaas(ctx, v.client, newPaasns.Spec.Paas)
	// This will not occur.
	// if err != nil {
	// 	return w, &field.Error{
	// 		Type:     field.ErrorTypeNotSupported,
	// 		Field:    field.NewPath("spec").Child("paas").String(),
	// 		BadValue: newPaasns.Spec.Paas,
	// 		Detail:   fmt.Sprintf("paas %s does not exist", newPaasns.Spec.Paas),
	// 	}
	// }

	if !reflect.DeepEqual(oldPaasns.Spec.Groups, newPaasns.Spec.Groups) {
		// Raise errors when groups are not in paas
		errs = append(errs, compareGroups(newPaasns.Spec.Groups, paas.Spec.Groups.Keys())...)
	}

	// Err when an sshSecret can't be decrypted
	if !reflect.DeepEqual(oldPaasns.Spec.SshSecrets, newPaasns.Spec.SshSecrets) {
		var validated validatedSecrets
		// We don't have to validate what is in the Paas (already validated by Paas webhook)
		validated.appendFromPaas(*paas)
		// We don't have to validate what is in the previous PaasNs definition (already validated before)
		validated.appendFromPaasNS(*oldPaasns)
		getRsaFunc := func() (cr *crypt.Crypt, err error) {
			return getRsa(ctx, v.client, newPaasns.Spec.Paas)
		}
		errs = append(errs, validated.compareSecrets(newPaasns.Spec.SshSecrets, getRsaFunc)...)
	}

	if len(errs) > 0 {
		return w, errs.ToAggregate()
	}
	return w, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type PaasNS.
func (v *PaasNSCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func getPaas(ctx context.Context, c client.Client, name string) (paas *v1alpha1.Paas, err error) {
	paas = &v1alpha1.Paas{}
	err = c.Get(ctx, client.ObjectKey{
		Name: name,
	}, paas)
	if err != nil {
		err = fmt.Errorf("failed to get paas: %w", err)
		_, logger := logging.GetLogComponent(ctx, "webhook_getPaas")
		logger.Error().Msg(err.Error())
		return nil, err
	}
	return paas, nil
}

// compareGroups is a helper function to compare the list of groups in a PaasNS against the list in the Paas
func compareGroups(subGroups []string, superGroups []string) (errs field.ErrorList) {
	uqSuperGroups := make(map[string]bool)
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

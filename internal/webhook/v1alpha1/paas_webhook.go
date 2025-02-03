/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"
	"fmt"

	apiv1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func setRequestLogger(ctx context.Context, obj client.Object) (context.Context, *zerolog.Logger) {
	logger := log.With().
		Any("webhook", obj.GetObjectKind().GroupVersionKind()).
		Dict("object", zerolog.Dict().
			Str("name", obj.GetName()).
			Str("namespace", obj.GetNamespace()),
		).
		Str("requestId", uuid.NewString()).
		Logger()

	return logger.WithContext(ctx), &logger
}

// SetupPaasWebhookWithManager registers the webhook for Paas in the manager.
func SetupPaasWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&apiv1alpha1.Paas{}).
		WithValidator(&PaasCustomValidator{client: mgr.GetClient()}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
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
	paas, ok := obj.(*apiv1alpha1.Paas)
	if !ok {
		return nil, fmt.Errorf("expected a Paas object but got %T", obj)
	}
	ctx, logger := setRequestLogger(ctx, paas)
	logger.Info().Msg("starting validation webhook for creation")

	config, err := v.getConfig(ctx)
	if err != nil {
		logger.Err(err).Send()
		return nil, err
	}

	for name := range paas.Spec.Capabilities {
		if _, ok := config.Capabilities[name]; !ok {
			return nil, fmt.Errorf("capability %s not configured", name)
		}
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	paas, ok := newObj.(*apiv1alpha1.Paas)
	if !ok {
		return nil, fmt.Errorf("expected a Paas object for the newObj but got %T", newObj)
	}
	_, logger := setRequestLogger(ctx, paas)
	logger.Info().Msg("starting validation webhook for update")

	// TODO(portly-halicore-76): fill in your validation logic upon object update.

	return nil, nil
}

// TODO(portly-halicore-76): determine whether this can be left out
// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Paas.
func (v *PaasCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	paas, ok := obj.(*apiv1alpha1.Paas)
	if !ok {
		return nil, fmt.Errorf("expected a Paas object but got %T", obj)
	}
	_, logger := setRequestLogger(ctx, paas)
	logger.Info().Msg("starting validation webhook for deletion")

	// TODO(portly-halicore-76): fill in your validation logic upon object deletion.

	return nil, nil
}

func (v *PaasCustomValidator) getConfig(ctx context.Context) (*apiv1alpha1.PaasConfigSpec, error) {
	configs := &apiv1alpha1.PaasConfigList{}
	err := v.client.List(ctx, configs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve PaasConfig list: %w", err)
	} else if len(configs.Items) != 1 {
		return nil, fmt.Errorf("invalid number of PaasConfigs: %d", len(configs.Items))
	}

	return &configs.Items[0].Spec, nil
}

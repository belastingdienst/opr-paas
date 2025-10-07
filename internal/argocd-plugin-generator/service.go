/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"context"
	"errors"
	"fmt"

	"github.com/belastingdienst/opr-paas/v3/internal/argocd-plugin-generator/fields"
	"github.com/belastingdienst/opr-paas/v3/internal/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/belastingdienst/opr-paas/v3/internal/config"

	"github.com/belastingdienst/opr-paas/v3/internal/templating"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
)

// Service provides the business logic for the plug-in generator.
//
// It encapsulates the Kubernetes controller-runtime client so that
// it can interact with cluster resources (e.g., list or get custom
// resources) as part of processing incoming plug-in requests.
type Service struct {
	kclient client.Client
}

// NewService creates a new Service instance.
//
// The provided controller-runtime Client will be used to read or
// modify Kubernetes objects. Typically, this client is injected
// by the controller manager and is backed by the shared informer
// cache for efficiency.
func NewService(kclient client.Client) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, logger := logging.GetLogComponent(ctx, logging.PluginGeneratorComponent)
	logger.Debug().Msg("New Service")
	return &Service{kclient: kclient}
}

// Generate returns a generated []map[string]interface{} based on the provided map[string]interface. The input map
// should contain a key: "capability" which stands for the capability, for which a map of parameters is generated.
// in case the input param is missing, or the generation fails, an error is returned.
func (s *Service) Generate(params map[string]interface{}) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, logger := logging.GetLogComponent(ctx, logging.PluginGeneratorComponent)

	var paasList v1alpha2.PaasList
	if err := s.kclient.List(ctx, &paasList); err != nil {
		logger.Error().AnErr("error", err).Msg("List error")
		return nil, err
	}

	logger.Debug().Int("num_paases", len(paasList.Items)).Msg("ArgoCD plugin cap")
	capName, ok := params["capability"].(string)
	if !ok || capName == "" {
		logger.Error().Str("name", capName).Msg("invalid capability param")
		return nil, errors.New("missing or invalid capability param")
	}

	var results []map[string]interface{}
	myConfig, err := config.GetConfig(ctx, s.kclient)
	if err != nil {
		logger.Error().AnErr("error", err).Msg("GetConfig error")
		return nil, err
	}
	for _, paas := range paasList.Items {
		var elements fields.Elements
		elements, err = capElementsFromPaas(ctx, &paas, capName, myConfig)
		if err != nil {
			logger.Error().Str("paas_name", paas.Name).AnErr("error", err).Msg("failed to generate elements")
			return nil, err // return error to caller
		}
		if elements == nil {
			continue
		}

		strMap := elements.GetElementsAsStringMap()

		inf := make(map[string]interface{}, len(strMap))
		for k, v := range strMap {
			inf[k] = v
		}
		logger.Debug().Str("paas_name", paas.Name).Int("num_elements", len(inf)).Msg("added paas")
		results = append(results, inf)
	}

	return results, nil
}

func capElementsFromPaas(
	ctx context.Context,
	paas *v1alpha2.Paas,
	capName string,
	paasConfig v1alpha2.PaasConfig,
) (elements fields.Elements, err error) {
	_, componentLogger := logging.GetLogComponent(ctx, logging.PluginGeneratorComponent)
	logger := componentLogger.With().Str("paas", paas.Name).Str("capability", capName).Logger()
	templater := templating.NewTemplater(*paas, paasConfig)
	capConfig, exists := paasConfig.Spec.Capabilities[capName]
	if !exists {
		logger.Error().Msg("capability is not configured")
		return nil, fmt.Errorf("capability %s is not configured", capName)
	}
	templatedElements, err := applyCustomFieldTemplates(capConfig.CustomFields, templater)
	if err != nil {
		logger.Error().AnErr("error", err).Msg("templating custom fields failed")
		return nil, err
	}

	capability, exists := paas.Spec.Capabilities[capName]
	if !exists {
		logger.Debug().Msg("capability not enabled")
		return nil, nil
	}

	capElements, err := capability.CapExtraFields(paasConfig.Spec.Capabilities[capName].CustomFields)
	if err != nil {
		logger.Error().AnErr("error", err).Msg("getting capability custom fields failed")
		return nil, err
	}
	elements = templatedElements.AsFieldElements().Merge(capElements)

	for name, tpl := range paasConfig.Spec.Templating.GenericCapabilityFields {
		result, templateErr := templater.TemplateToMap(name, tpl)
		if templateErr != nil {
			logger.Error().Str("template", tpl).AnErr("error", templateErr).Msg("templating failed")
			return nil, fmt.Errorf("failed to run template %s", tpl)
		}
		for key, value := range result {
			elements[key] = value
		}
	}

	elements["paas"] = paas.Name
	logger.Debug().Str("paas", paas.Name).Int("num_elements", len(elements)).Msg("returning elements")
	return elements, nil
}

func applyCustomFieldTemplates(
	ccfields map[string]v1alpha2.ConfigCustomField,
	templater templating.Templater[v1alpha2.Paas, v1alpha2.PaasConfig, v1alpha2.PaasConfigSpec],
) (templating.TemplateResult, error) {
	var result templating.TemplateResult

	for name, fieldConfig := range ccfields {
		if fieldConfig.Template != "" {
			fieldResult, err := templater.TemplateToMap(name, fieldConfig.Template)
			if err != nil {
				return nil, err
			}
			result = result.Merge(fieldResult)
		}
	}

	return result, nil
}

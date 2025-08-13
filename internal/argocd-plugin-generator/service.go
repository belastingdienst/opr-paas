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
	"github.com/rs/zerolog/log"
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
	return &Service{kclient: kclient}
}

// Generate returns a generated []map[string]interface{} based on the provided map[string]interface. The input map
// should contain a key: "capability" which stands for the capability, for which a map of parameters is generated.
// in case the input param is missing, or the generation fails, an error is returned.
func (s *Service) Generate(params map[string]interface{}, appSetName string) ([]map[string]interface{}, error) {
	ctx := context.Background()

	var paasList v1alpha2.PaasList
	if err := s.kclient.List(ctx, &paasList); err != nil {
		return nil, err
	}

	log.Debug().Msgf("ArgoCD plugin cap, listed: %v Paases", len(paasList.Items))
	capName, ok := params["capability"].(string)
	if !ok || capName == "" {
		return nil, errors.New("missing or invalid capability param")
	}

	var results []map[string]interface{}
	for _, paas := range paasList.Items {
		if _, ok = paas.Spec.Capabilities[capName]; !ok {
			continue
		}

		elements, err := capElementsFromPaas(&paas, capName)
		if err != nil {
			continue // skip failed ones
		}

		strMap := elements.GetElementsAsStringMap()

		inf := make(map[string]interface{}, len(strMap))
		for k, v := range strMap {
			inf[k] = v
		}
		results = append(results, inf)
	}

	return results, nil
}

func capElementsFromPaas(
	paas *v1alpha2.Paas,
	capName string,
) (elements fields.Elements, err error) {
	myConfig := config.GetConfig()
	templater := templating.NewTemplater(*paas, myConfig)
	capConfig := myConfig.Spec.Capabilities[capName]
	templatedElements, err := applyCustomFieldTemplates(capConfig.CustomFields, templater)
	if err != nil {
		return nil, err
	}

	capability := paas.Spec.Capabilities[capName]

	capElements, err := capability.CapExtraFields(myConfig.Spec.Capabilities[capName].CustomFields)
	if err != nil {
		return nil, err
	}
	elements = templatedElements.AsFieldElements().Merge(capElements)

	for name, tpl := range myConfig.Spec.Templating.GenericCapabilityFields {
		result, templateErr := templater.TemplateToMap(name, tpl)
		if templateErr != nil {
			return nil, fmt.Errorf("failed to run template %s", tpl)
		}
		for key, value := range result {
			elements[key] = value
		}
	}

	elements["paas"] = paas.Name
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

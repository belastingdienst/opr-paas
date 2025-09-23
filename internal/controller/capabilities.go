/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

// Package controller has all logic for reconciling Paas resources
package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/belastingdienst/opr-paas/v3/internal/argocd-plugin-generator/fields"
	"github.com/belastingdienst/opr-paas/v3/internal/logging"
	appv1 "github.com/belastingdienst/opr-paas/v3/internal/stubs/argoproj/v1alpha1"
	"github.com/belastingdienst/opr-paas/v3/internal/templating"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func clearGenerators(generators []appv1.ApplicationSetGenerator) (clean []appv1.ApplicationSetGenerator) {
	for _, generator := range generators {
		if generator.List == nil {
			// Not a list generator, not introduced by paas operator, we should preserve
			clean = append(clean, generator)
		} else if len(generator.List.Elements) > 0 {
			clean = append(clean, generator)
		}
	}
	return clean
}

func getListGen(generators []appv1.ApplicationSetGenerator) *appv1.ApplicationSetGenerator {
	for _, generator := range generators {
		if len(generator.List.Elements) > 0 {
			return &generator
		}
	}
	return nil
}

// ensureAppSetCap ensures a list entry in the AppSet for each capability
func (r *PaasReconciler) ensureAppSetCaps(
	ctx context.Context,
	paas *v1alpha2.Paas,
) error {
	myConfig, err := getConfigFromContext(ctx)
	if err != nil {
		return err
	}
	for capName := range paas.Spec.Capabilities {
		if _, exists := myConfig.Spec.Capabilities[capName]; !exists {
			return errors.New("capability not configured")
		}

		if err = r.ensureAppSetCap(ctx, paas, capName); err != nil {
			return err
		}
	}

	return nil
}

func capElementsFromPaas(
	ctx context.Context,
	paas *v1alpha2.Paas,
	capName string,
) (elements fields.Elements, err error) {
	paasConfig, err := getConfigFromContext(ctx)
	if err != nil {
		return nil, err
	}
	templater := templating.NewTemplater(*paas, paasConfig)
	capConfig := paasConfig.Spec.Capabilities[capName]
	templatedElements, err := applyCustomFieldTemplates(capConfig.CustomFields, templater)
	if err != nil {
		return nil, err
	}

	capability := paas.Spec.Capabilities[capName]

	capElements, err := capability.CapExtraFields(paasConfig.Spec.Capabilities[capName].CustomFields)
	if err != nil {
		return nil, err
	}
	elements = templatedElements.AsFieldElements().Merge(capElements)

	for name, tpl := range paasConfig.Spec.Templating.GenericCapabilityFields {
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

// ensureAppSetCap ensures a list entry in the AppSet for the capability
func (r *PaasReconciler) ensureAppSetCap(
	ctx context.Context,
	paas *v1alpha2.Paas,
	capName string,
) error {
	var err error
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerCapabilitiesComponent)
	logger.Info().Msgf("reconciling %s Applicationset", capName)
	myConfig, err := getConfigFromContext(ctx)
	if err != nil {
		return err
	}
	namespacedName := myConfig.Spec.CapabilityK8sName(capName)
	if !reflect.DeepEqual(namespacedName, types.NamespacedName{}) {
		appSet := &appv1.ApplicationSet{}
		err = r.Get(ctx, namespacedName, appSet)
		var entries fields.Entries
		var listGen *appv1.ApplicationSetGenerator
		if err != nil {
			// Applicationset does not exist
			return err
		}

		elements, err2 := capElementsFromPaas(ctx, paas, capName)
		if err2 != nil {
			return err
		}
		patch := client.MergeFrom(appSet.DeepCopy())
		if listGen = getListGen(appSet.Spec.Generators); listGen == nil {
			// create the list
			listGen = &appv1.ApplicationSetGenerator{
				List: &appv1.ListGenerator{},
			}
			appSet.Spec.Generators = append(appSet.Spec.Generators, *listGen)
			entries = fields.Entries{
				paas.Name: elements,
			}
		} else if entries, err = fields.EntriesFromJSON(listGen.List.Elements); err != nil {
			return err
		} else {
			entry := elements
			entries[entry.Key()] = entry
		}
		jsonentries, err3 := entries.AsJSON()
		if err3 != nil {
			return err
		}
		listGen.List.Elements = jsonentries

		appSet.Spec.Generators = clearGenerators(appSet.Spec.Generators)
		return r.Patch(ctx, appSet, patch)
	}

	// According to config, we don't manage ListGenerators, therefore we don't return an error
	return nil
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

// finalizeAppSetCap ensures the list entries in the AppSet is removed for the capability of this PaasNs
func (r *PaasReconciler) finalizeAppSetCap(
	ctx context.Context,
	paasName string,
	capName string,
) error {
	as := &appv1.ApplicationSet{}
	myConfig, err := getConfigFromContext(ctx)
	if err != nil {
		return err
	}
	asNamespacedName := myConfig.Spec.CapabilityK8sName(capName)
	if !reflect.DeepEqual(asNamespacedName, types.NamespacedName{}) {
		err = r.Get(ctx, asNamespacedName, as)
		var entries fields.Entries
		var listGen *appv1.ApplicationSetGenerator
		if err != nil {
			// Applicationset does not exist
			return nil
		}
		patch := client.MergeFrom(as.DeepCopy())
		if listGen = getListGen(as.Spec.Generators); listGen == nil {
			// no need to create the list
			return nil
		} else if entries, err = fields.EntriesFromJSON(listGen.List.Elements); err != nil {
			return err
		}
		delete(entries, paasName)
		var jsonentries []v1.JSON
		jsonentries, err = entries.AsJSON()
		if err != nil {
			return err
		}
		listGen.List.Elements = jsonentries
		return r.Patch(ctx, as, patch)
	}
	// According to config, we don't manage ListGenerators, therefore we don't return an error
	return nil
}

// finalizeAllAppSetCaps removes this paas from all capability appsets
func (r *PaasReconciler) finalizeAllAppSetCaps(
	ctx context.Context,
	paas *v1alpha2.Paas,
) error {
	paasWithoutCaps := paas.DeepCopy()
	paasWithoutCaps.Spec.Capabilities = nil
	return r.finalizeDisabledAppSetCaps(ctx, paasWithoutCaps)
}

// finalizeAppSetCaps removes this paas from all capability appsets that are not enabled in this paas
func (r *PaasReconciler) finalizeDisabledAppSetCaps(
	ctx context.Context,
	paas *v1alpha2.Paas,
) error {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerCapabilitiesComponent)
	myConfig, err := getConfigFromContext(ctx)
	if err != nil {
		return err
	}
	for capName := range myConfig.Spec.Capabilities {
		logger.Info().Msgf("reconciling %s Applicationset", capName)
		if _, exists := paas.Spec.Capabilities[capName]; exists {
			continue
		}
		err = r.finalizeAppSetCap(ctx, paas.Name, capName)
		if err != nil {
			return err
		}
	}
	return nil
}

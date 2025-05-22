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
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/fields"
	"github.com/belastingdienst/opr-paas/internal/logging"
	appv1 "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/templating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func splitToService(paasName string) (string, string) {
	parts := strings.SplitN(paasName, "-", 3)
	if len(parts) < 2 {
		return paasName, ""
	}
	return parts[0], parts[1]
}

// ensureAppSetCap ensures a list entry in the AppSet for each capability
func (r *PaasReconciler) ensureAppSetCaps(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	paasConfigSpec := config.GetConfig().Spec
	for capName := range paas.Spec.Capabilities {
		if _, exists := paasConfigSpec.Capabilities[capName]; !exists {
			return errors.New("capability not configured")
		}
		// Only do this when enabled
		capability := paas.Spec.Capabilities[capName]
		if enabled := capability.IsEnabled(); enabled {
			if err := r.ensureAppSetCap(ctx, paas, capName); err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureAppSetCap ensures a list entry in the AppSet for the capability
func (r *PaasReconciler) ensureAppSetCap(
	ctx context.Context,
	paas *v1alpha1.Paas,
	capName string,
) error {
	var err error
	var elements fields.Elements
	// See if AppSet exists raise error if it doesn't
	namespacedName := config.GetConfig().Spec.CapabilityK8sName(capName)
	appSet := &appv1.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Applicationset",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
	}
	ctx, logger := logging.GetLogComponent(ctx, "appset")
	logger.Info().Msgf("reconciling %s Applicationset", capName)
	err = r.Get(ctx, namespacedName, appSet)
	var entries fields.Entries
	var listGen *appv1.ApplicationSetGenerator
	if err != nil {
		// Applicationset does not exist
		return err
	}

	myConfig := config.GetConfig()
	templater := templating.NewTemplater(*paas, myConfig)
	spec := myConfig.GetSpec()
	capConfig := spec.Capabilities[capName]
	templatedElements, err := applyCustomFieldTemplates(capConfig.CustomFields, templater)
	if err != nil {
		return err
	}

	capability := paas.Spec.Capabilities[capName]

	// FIXME(hikarukin): temporarily convert v2 to v1
	cfs := make(map[string]v1alpha1.ConfigCustomField, len(myConfig.Spec.Capabilities[capName].CustomFields))
	for key, val := range myConfig.Spec.Capabilities[capName].CustomFields {
		cfs[key] = val.ConvertTo()
	}

	capElements, err := capability.CapExtraFields(cfs)
	if err != nil {
		return err
	}
	elements = templatedElements.AsFieldElements().Merge(capElements)
	service, subService := splitToService(paas.Name)
	elements["requestor"] = paas.Spec.Requestor
	elements["paas"] = paas.Name
	elements["service"] = service
	elements["subservice"] = subService
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
	jsonentries, err := entries.AsJSON()
	if err != nil {
		return err
	}
	listGen.List.Elements = jsonentries

	appSet.Spec.Generators = clearGenerators(appSet.Spec.Generators)
	return r.Patch(ctx, appSet, patch)
}

func applyCustomFieldTemplates(
	ccfields map[string]v1alpha2.ConfigCustomField,
	templater templating.Templater[v1alpha1.Paas, v1alpha2.PaasConfig, v1alpha2.PaasConfigSpec],
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
	// See if AppSet exists raise error if it doesn't
	as := &appv1.ApplicationSet{}
	asNamespacedName := config.GetConfig().Spec.CapabilityK8sName(capName)
	err := r.Get(ctx, asNamespacedName, as)
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
	jsonentries, err := entries.AsJSON()
	if err != nil {
		return err
	}
	listGen.List.Elements = jsonentries
	return r.Patch(ctx, as, patch)
}

// finalizeAppSetCaps ensures all capabilities that exist are removed from the Capability Applicationsets
func (r *PaasReconciler) finalizeAppSetCaps(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	ctx, logger := logging.GetLogComponent(ctx, "Applicationsets")
	for capName := range config.GetConfig().Spec.Capabilities {
		logger.Info().Msgf("reconciling %s Applicationset", capName)
		err := r.finalizeAppSetCap(ctx, paas.Name, capName)
		if err != nil {
			return err
		}
	}
	return nil
}

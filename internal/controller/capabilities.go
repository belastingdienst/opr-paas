/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	appv1 "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	"github.com/rs/zerolog/log"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Elements represents all key, value pars for one entry in the list of the listgenerator
type Elements map[string]string

func ElementsFromJSON(raw []byte) (Elements, error) {
	newElements := make(Elements)
	if err := json.Unmarshal(raw, &newElements); err != nil {
		return nil, err
	} else {
		return newElements, nil
	}
}

func (es Elements) AsString() string {
	var l []string
	for key, value := range es {
		l = append(l, fmt.Sprintf("'%s': '%s'", key, strings.ReplaceAll(value, "'", "\\'")))
	}
	return fmt.Sprintf("{ %s }", strings.Join(l, ", "))
}

func (es Elements) AsJSON() ([]byte, error) {
	return json.Marshal(es)
}

func (es Elements) Key() string {
	if key, exists := es["paas"]; exists {
		return key
	}
	return ""
}

// Entries represents all entries in the list of the listgenerator
// This is a map so that values are unique, the key is the paas entry
type Entries map[string]Elements

func (en Entries) AsString() string {
	var l []string
	for key, value := range en {
		l = append(l, fmt.Sprintf("'%s': %s", key, value.AsString()))
	}
	return fmt.Sprintf("{ %s }", strings.Join(l, ", "))
}

func (en Entries) AsJSON() ([]apiextensionsv1.JSON, error) {
	list := []apiextensionsv1.JSON{}
	for _, entry := range en {
		if data, err := entry.AsJSON(); err != nil {
			return nil, err
		} else {
			list = append(list, apiextensionsv1.JSON{Raw: data})
		}
	}
	return list, nil
}

func EntriesFromJSON(data []apiextensionsv1.JSON) (Entries, error) {
	e := Entries{}
	for _, raw := range data {
		if entry, err := ElementsFromJSON(raw.Raw); err != nil {
			return nil, err
		} else {
			key := entry.Key()
			if key != "" {
				e[key] = entry
			} else {
				// weird, this entry does not have a paas key, let's preserve, but put aside
				e[string(raw.Raw)] = entry
			}
		}
	}
	return e, nil
}

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
	config := GetConfig()
	for capName := range paas.Spec.Capabilities {
		if _, exists := config.Capabilities[capName]; !exists {
			return fmt.Errorf("Capability not configured")
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
	var fields Elements
	// See if AppSet exists raise error if it doesn't
	namespacedName := GetConfig().CapabilityK8sName(capName)
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
	ctx = setLogComponent(ctx, "appset")
	log.Ctx(ctx).Info().Msgf("reconciling %s Applicationset", capName)
	err = r.Get(ctx, namespacedName, appSet)
	var entries Entries
	var listGen *appv1.ApplicationSetGenerator
	if err != nil {
		// Applicationset does not exist
		return err
	}

	capability := paas.Spec.Capabilities[capName]
	if fields, err = capability.CapExtraFields(GetConfig().Capabilities[capName].CustomFields); err != nil {
		return err
	}
	service, subService := splitToService(paas.Name)
	fields["requestor"] = paas.Spec.Requestor
	fields["paas"] = paas.Name
	fields["service"] = service
	fields["subservice"] = subService
	patch := client.MergeFrom(appSet.DeepCopy())
	if listGen = getListGen(appSet.Spec.Generators); listGen == nil {
		// create the list
		listGen = &appv1.ApplicationSetGenerator{
			List: &appv1.ListGenerator{},
		}
		appSet.Spec.Generators = append(appSet.Spec.Generators, *listGen)
		entries = Entries{
			paas.Name: fields,
		}
	} else if entries, err = EntriesFromJSON(listGen.List.Elements); err != nil {
		return err
	} else {
		entry := fields
		entries[entry.Key()] = entry
	}
	// log.Info(fmt.Sprintf("entries: %s", entries.AsString()))
	if json, err := entries.AsJSON(); err != nil {
		return err
	} else {
		listGen.List.Elements = json
	}

	appSet.Spec.Generators = clearGenerators(appSet.Spec.Generators)
	return r.Patch(ctx, appSet, patch)
}

// finalizeAppSetCaps ensures the list entries in the AppSets are removed for each capability in the Paas
func (r *PaasReconciler) finalizeAppSetCaps(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	for capName := range paas.Spec.Capabilities {
		if err := r.finalizeAppSetCap(ctx, paas, capName); err != nil {
			return err
		}
	}
	return nil
}

// finalizeAppSetCap ensures the list entry for the capability is removed
func (r *PaasReconciler) finalizeAppSetCap(
	ctx context.Context,
	paas *v1alpha1.Paas,
	capability string,
) error {
	// See if AppSet exists raise error if it doesn't
	as := &appv1.ApplicationSet{}
	asNamespacedName := GetConfig().CapabilityK8sName(capability)
	ctx = setLogComponent(ctx, "appset")
	log.Ctx(ctx).Info().Msgf("reconciling %s Applicationset", capability)
	err := r.Get(ctx, asNamespacedName, as)
	var entries Entries
	var listGen *appv1.ApplicationSetGenerator
	if err != nil {
		// Applicationset does not exixt
		return nil
	}
	patch := client.MergeFrom(as.DeepCopy())
	if listGen = getListGen(as.Spec.Generators); listGen == nil {
		// no need to create the list
		return nil
	} else if entries, err = EntriesFromJSON(listGen.List.Elements); err != nil {
		return err
	} else {
		delete(entries, paas.Name)
	}
	if json, err := entries.AsJSON(); err != nil {
		return err
	} else {
		listGen.List.Elements = json
	}

	return r.Patch(ctx, as, patch)
}

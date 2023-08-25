package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	appv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

func entryFromPaas(paas *v1alpha1.Paas) Elements {
	service, subService := splitToService(paas.Name)
	return Elements{
		"oplosgroep": paas.Spec.Oplosgroep,
		"paas":       paas.Name,
		"service":    service,
		"subservice": subService,
	}
}

// ensureAppSetCap ensures a list entry in the AppSet voor the capability
func (r *PaasReconciler) ensureAppSetCap(
	ctx context.Context,
	paas *v1alpha1.Paas,
	capability string,
) error {
	// See if AppSet exists raise error if it doesn't
	as := &appv1.ApplicationSet{}
	asNamespacedName := getConfig().CapabilityK8sName(capability)
	logger := getLogger(ctx, paas, "AppSet", asNamespacedName.String())
	logger.Info(fmt.Sprintf("Reconciling %s Applicationset", capability))
	err := r.Get(ctx, asNamespacedName, as)
	//groups := NewGroups().AddFromStrings(paas.Spec.LdapGroups)
	var entries Entries
	var listGen *appv1.ApplicationSetGenerator
	if err != nil {
		// Applicationset does not exixt
		return err
	} else if listGen = getListGen(as.Spec.Generators); listGen == nil {
		// create the list
		listGen = &appv1.ApplicationSetGenerator{
			List: &appv1.ListGenerator{},
		}
		as.Spec.Generators = append(as.Spec.Generators, *listGen)
		entries = Entries{
			paas.Name: entryFromPaas(paas),
		}
	} else if entries, err = EntriesFromJSON(listGen.List.Elements); err != nil {
		return err
	} else {
		entry := entryFromPaas(paas)
		entries[entry.Key()] = entry
	}
	// log.Info(fmt.Sprintf("entries: %s", entries.AsString()))
	if json, err := entries.AsJSON(); err != nil {
		return err
	} else {
		// log.Info(fmt.Sprintf("json: %v", json))
		// log.Info(fmt.Sprintf("json: %v", listGen))
		// log.Info(fmt.Sprintf("json: %v", listGen.List))
		listGen.List.Elements = json
	}

	return r.Update(ctx, as)
}

// ensureAppSetCap ensures a list entry in the AppSet voor the capability
func (r *PaasReconciler) EnsureAppSetCaps(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	type cap struct {
		name    string
		enabled bool
	}
	for _, c := range []cap{
		{name: "argocd", enabled: paas.Spec.Capabilities.ArgoCD.Enabled},
		{name: "tekton", enabled: paas.Spec.Capabilities.CI.Enabled},
		{name: "grafana", enabled: paas.Spec.Capabilities.Grafana.Enabled},
		{name: "sso", enabled: paas.Spec.Capabilities.SSO.Enabled},
	} {
		if c.enabled {
			if err := r.ensureAppSetCap(ctx, paas, c.name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *PaasReconciler) finalizeAppSetCap(
	ctx context.Context,
	paas *v1alpha1.Paas,
	capability string,
) error {
	// See if AppSet exists raise error if it doesn't
	as := &appv1.ApplicationSet{}
	asNamespacedName := getConfig().CapabilityK8sName(capability)
	logger := getLogger(ctx, paas, "AppSet", asNamespacedName.String())
	logger.Info(fmt.Sprintf("Reconciling %s Applicationset", capability))
	err := r.Get(ctx, asNamespacedName, as)
	//groups := NewGroups().AddFromStrings(paas.Spec.LdapGroups)
	var entries Entries
	var listGen *appv1.ApplicationSetGenerator
	if err != nil {
		// Applicationset does not exixt
		return nil
	} else if listGen = getListGen(as.Spec.Generators); listGen == nil {
		// no need to create the list
		return nil
	} else if entries, err = EntriesFromJSON(listGen.List.Elements); err != nil {
		return err
	} else {
		entry := entryFromPaas(paas)
		delete(entries, entry.Key())
	}
	if json, err := entries.AsJSON(); err != nil {
		return err
	} else {
		listGen.List.Elements = json
	}

	return r.Update(ctx, as)
}

// ensureAppSetCap ensures a list entry in the AppSet voor the capability
func (r *PaasReconciler) FinalizeAppSetCaps(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	for _, c := range []string{
		"argocd",
		"ci",
		"grafana",
		"sso",
	} {
		if err := r.finalizeAppSetCap(ctx, paas, c); err != nil {
			return err
		}
	}
	return nil
}

package controller

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// namespaceDef is an internal struct so that we can collect all namespace info regarding this paas once,
// and reuse for every reconciliation for a subresources (.e.a. namespaces, secrets, rolebindings, etc.).
type namespaceDef struct {
	nsName    string
	paasns    *v1alpha1.PaasNS
	capName   string
	capConfig v1alpha2.ConfigCapability
	quota     string
	groups    []string
	secrets   map[string]string
}

type namespaceDefs map[string]namespaceDef

// Helper to create a base namespaceDef
func newNamespaceDef(nsName, quota string, groups []string, secrets map[string]string) namespaceDef {
	return namespaceDef{
		nsName:  nsName,
		quota:   quota,
		groups:  groups,
		secrets: secrets,
	}
}

// Helper to create a namespaceDef from a PaasNS
func newNamespaceDefFromPaasNS(nsName string, paasns *v1alpha1.PaasNS,
	quota string, defaultGroups []string, secrets map[string]string) namespaceDef {
	groups := defaultGroups
	if len(paasns.Spec.Groups) > 0 {
		groups = paasns.Spec.Groups
	}
	if paasns.Spec.SSHSecrets != nil {
		secrets = mergeSecrets(secrets, paasns.Spec.SSHSecrets)
	}
	return namespaceDef{
		nsName:  nsName,
		paasns:  paasns,
		quota:   quota,
		groups:  groups,
		secrets: secrets,
	}
}

// Helper to merge secrets
func mergeSecrets(base, override map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		merged[k] = v
	}
	return merged
}

// paasNSsFromNs gets all PaasNs objects from a namespace and returns a map of all paasNS's.
// Key of the map is based on the namespaced name of the PaasNS, so that we have uniqueness
// paasNSsFromNs runs recursively, to collect all PaasNS's in the namespaces of founds PaasNS namespaces too.
func (r *PaasReconciler) paasNSsFromNs(ctx context.Context, ns string) map[string]v1alpha1.PaasNS {
	nss := map[string]v1alpha1.PaasNS{}
	pnsList := &v1alpha1.PaasNSList{}
	if err := r.List(ctx, pnsList, &client.ListOptions{Namespace: ns}); err != nil {
		// In this case panic is ok, since this situation can only occur when either k8s is down,
		// or permissions are insufficient. Both cases we should not continue executing code...
		panic(err)
	}
	for _, pns := range pnsList.Items {
		nsName := pns.NamespaceName()
		nss[nsName] = pns
		for key, value := range r.paasNSsFromNs(ctx, nsName) {
			nss[key] = value
		}
	}
	return nss
}

func (r *PaasReconciler) nsDefsFromPaasNamespaces(ctx context.Context, paas *v1alpha1.Paas,
	paasGroups []string) namespaceDefs {
	result := namespaceDefs{}
	for _, namespace := range paas.Spec.Namespaces {
		fullNsName := join(paas.Name, namespace)
		base := newNamespaceDef(fullNsName, paas.Name, paasGroups, paas.Spec.SSHSecrets)
		result[base.nsName] = base

		for nsName, paasns := range r.paasNSsFromNs(ctx, base.nsName) {
			ns := newNamespaceDefFromPaasNS(nsName, &paasns, paas.Name, paasGroups, paas.Spec.SSHSecrets)
			result[ns.nsName] = ns
		}
	}
	return result
}

func (r *PaasReconciler) paasCapabilityNss(ctx context.Context, paas *v1alpha1.Paas,
	paasGroups []string) (namespaceDefs, error) {
	result := namespaceDefs{}
	capsConfig := config.GetConfig().Spec.Capabilities

	for capName, capDef := range paas.Spec.Capabilities {
		if !capDef.Enabled {
			continue
		}
		capConfig, ok := capsConfig[capName]
		if !ok {
			return nil, fmt.Errorf("capability %s is not in PaasConfig", capName)
		}
		capNS := join(paas.Name, capName)
		quota := capNS
		if capConfig.QuotaSettings.Clusterwide {
			quota = clusterWideQuotaName(capName)
		}
		secrets := mergeSecrets(paas.Spec.SSHSecrets, capDef.SSHSecrets)
		base := namespaceDef{
			nsName:    capNS,
			capName:   capName,
			capConfig: capConfig,
			quota:     quota,
			groups:    paasGroups,
			secrets:   secrets,
		}
		result[base.nsName] = base
		for nsName, paasns := range r.paasNSsFromNs(ctx, capNS) {
			ns := newNamespaceDefFromPaasNS(nsName, &paasns, paas.Name, paasGroups, paas.Spec.SSHSecrets)
			result[ns.nsName] = ns
		}
	}
	return result, nil
}

// nsFromPaas accepts a Paas and returns a list of all namespaceDefs managed by this Paas
// this is a combination of
// - all namespaces as defined in paas.spec.namespaces
// - all namespaces as required by paas.spec.capabilities
// - all namespaces as required by paasNS's belonging to this paas
func (r *PaasReconciler) nsDefsFromPaas(ctx context.Context, paas *v1alpha1.Paas) (namespaceDefs, error) {
	paasGroups := paas.Spec.Groups.Keys()
	nsDefs := namespaceDefs{}
	for _, ns := range r.nsDefsFromPaasNamespaces(ctx, paas, paasGroups) {
		nsDefs[ns.nsName] = ns
	}
	capNss, err := r.paasCapabilityNss(ctx, paas, paasGroups)
	if err != nil {
		return nil, err
	}
	for _, ns := range capNss {
		nsDefs[ns.nsName] = ns
	}
	return nsDefs, nil
}

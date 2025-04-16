package controller

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// namespaceDef is an internal struct so that we can collect all namespace info regarding this paas once,
// and reuse for every reconciliation for a subresources (.e.a. namespaces, secrets, rolebindings, etc.).
type namespaceDef struct {
	nsName    string
	paasns    *v1alpha1.PaasNS
	capName   string
	capConfig v1alpha1.ConfigCapability
	quota     string
	groups    []string
	secrets   map[string]string
}

type namespaceDefs []namespaceDef

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
		// Call myself (recursively)
		for key, value := range r.paasNSsFromNs(ctx, nsName) {
			nss[key] = value
		}
	}
	return nss
}

// nsFromPaas accepts a Paas and returns a list of all namespaceDefs managed by this Paas
// this is a combination of
// - all namespaces as defined in paas.spec.namespaces
// - all namespaces as required by paas.spec.capabilities
// - all namespaces as required by paasNS's belonging to this paas
func (r *PaasReconciler) nsDefsFromPaas(ctx context.Context, paas *v1alpha1.Paas) (namespaceDefs, error) {
	var finalNss namespaceDefs
	var paasGroups []string
	for key, paasGroup := range paas.Spec.Groups {
		if paasGroup.Query == "" {
			key = join(paas.Name, key)
		}
		paasGroups = append(paasGroups, key)
	}
	for _, nsName := range paas.Spec.Namespaces {
		ns := namespaceDef{
			nsName:  join(paas.Name, nsName),
			quota:   paas.Name,
			groups:  paasGroups,
			secrets: paas.Spec.SSHSecrets,
		}

		finalNss = append(finalNss, ns)
		for nsName, paasns := range r.paasNSsFromNs(ctx, ns.nsName) {
			ns = namespaceDef{
				nsName:  nsName,
				paasns:  &paasns,
				quota:   paas.Name,
				groups:  paasns.Spec.Groups,
				secrets: paas.Spec.SSHSecrets,
			}
			if len(paasns.Spec.Groups) == 0 {
				ns.groups = paasGroups
			}
			finalNss = append(finalNss, ns)
		}
	}
	capsConfig := config.GetConfig().Spec.Capabilities
	for capName, capDefinition := range paas.Spec.Capabilities {
		if capDefinition.Enabled != true {
			continue
		}
		capConfig, ok := capsConfig[capName]
		if !ok {
			return nil, fmt.Errorf("capability %s is not in PaasConfig", capName)
		}
		capNS := join(paas.Name, capName)
		quotaName := capNS
		if capConfig.QuotaSettings.Clusterwide {
			quotaName = clusterWideQuotaName(capName)
		}
		ns := namespaceDef{
			nsName:    capNS,
			capName:   capName,
			capConfig: capConfig,
			quota:     quotaName,
			groups:    paasGroups,
			secrets:   capDefinition.SSHSecrets,
		}
		finalNss = append(finalNss, ns)
		for nsName, paasns := range r.paasNSsFromNs(ctx, capNS) {
			ns = namespaceDef{
				nsName:  nsName,
				paasns:  &paasns,
				quota:   paas.Name,
				groups:  paasns.Spec.Groups,
				secrets: paas.Spec.SSHSecrets,
			}
			if len(paasns.Spec.Groups) == 0 {
				ns.groups = paasGroups
			}
			finalNss = append(finalNss, ns)
		}
	}
	return finalNss, nil
}

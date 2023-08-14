package controllers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
    DefaultClusterQuotaGroupName = "clusterquotagroup"
	DefaultNameSpace             = "gitops"
	DefaultApplicationsetName    = "???"
)

// CaasWhiteList returns a Namespaced object name which points to the
// Caas Whitelist where the ldap groupds should be defined
// Defaults point to kube-system.caaswhitelist, but can be overruled with
// the environment variables CAAS_WHITELIST_NAMESPACE and CAAS_WHITELIST_NAME
func CapabilityK8sName(capability string) (as types.NamespacedName) {
	if as.Namespace = os.Getenv("CAP_NAMESPACE"); as.Namespace == "" {
		as.Namespace = DefaultNameSpace
	}
	if as.Name = os.Getenv(fmt.Sprintf("CAP_%s_AS_NAME", strings.ToUpper(capability))); as.Name == "" {
		as.Name = fmt.Sprintf("%s-capability", DefaultApplicationsetName)
	}
	return as
}
func CapabilityClusterQuotaGroupName() string {
	if name = os.Getenv("CAP_CLUSTER_RESOURCE_QUOTA_NAME"); name != "" {
		return name
	}
	return DefaultClusterQuotaGroupName
}
func getLogger(
	ctx context.Context,
	paas *v1alpha1.Paas,
	kind string,
	name string,
) logr.Logger {
	fields := append(make([]interface{}, 0), "Paas", paas.Name, "Kind", kind)
	if name != "" {
		fields = append(fields, "Name", name)
	}

	return log.FromContext(ctx).WithValues(fields...)
}

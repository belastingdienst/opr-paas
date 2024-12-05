/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Definitions to manage status conditions
const (
	// TypeActivePaasConfig represents whether this is the PaasConfig being used by the Paas operator
	TypeActivePaasConfig = "Active"
	// TypeHasErrorsPaasConfig represents the status used when the custom resource reconciliation holds errors.
	TypeHasErrorsPaasConfig = "HasErrors"
	// TypeDegradedPaasConfig represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	TypeDegradedPaasConfig = "Degraded"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=paasconfig,scope=Cluster
type PaasConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PaasConfigSpec   `json:"spec,omitempty"`
	Status PaasConfigStatus `json:"status,omitempty"`
}

type PaasConfigSpec struct {
	// Deprecated: Will be replaced by a secretRef to overcome caching
	// Paths where the manager can find the decryptKeys to decrypt Paas'es
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Required
	DecryptKeyPaths []string `json:"decryptKeyPaths"`

	// Enable debug information generation or not
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	Debug bool `json:"debug,omitempty"`

	// A map with zero or more ConfigCapability
	// +kubebuilder:validation:Required
	Capabilities ConfigCapabilities `json:"capabilities"`

	// Deprecated: Whitelist code will be removed from the operator to make it more generic
	// A reference to a configmap containing a whitelist of LDAP groups to be synced using LDAP sync
	// +kubebuilder:validation:Required
	Whitelist NamespacedName `json:"whitelist"`

	// LDAP configuration for the operator to add to Groups
	// +kubebuilder:validation:Optional
	LDAP ConfigLdap `json:"ldap,omitempty"`

	// Deprecated: ArgoCD specific code will be removed from the operator
	// Permissions to set for ArgoCD instance
	// +kubebuilder:validation:Required
	ArgoPermissions ConfigArgoPermissions `json:"argopermissions,omitempty"`

	// Namespace in which ArgoCD applicationSets will be found for managing capabilities
	// +kubebuilder:default:=argocd
	// +kubebuilder:validation:Required
	AppSetNamespace string `json:"applicationset_namespace,omitempty"`

	// Label which is added to clusterquotas
	// +kubebuilder:default:=clusterquotagroup
	// +kubebuilder:validation:Optional
	QuotaLabel string `json:"quota_label,omitempty"`

	// Name of the label used to define who is the contact for this resource
	// +kubebuilder:default:=requestor
	// +kubebuilder:validation:Optional
	RequestorLabel string `json:"requestor_label,omitempty"`

	// Name of the label used to define by whom the resource is managed.
	// +kubebuilder:default:=argocd.argoproj.io/managed-by
	// +kubebuilder:validation:Optional
	ManagedByLabel string `json:"managed_by_label,omitempty"`

	// Deprecated: ArgoCD specific code will be removed from the operator
	// Name of an ApplicationSet to be set as ignored in the ArgoCD bootstrap Application
	// +kubebuilder:validation:Required
	ExcludeAppSetName string `json:"exclude_appset_name"`

	// Grant permissions to all groups according to config in configmap and role selected per group in paas.
	// +kubebuilder:validation:Optional
	RoleMappings ConfigRoleMappings `json:"rolemappings,omitempty"`
}

type NamespacedName struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
}

type ConfigRoleMappings map[string][]string

func (crm ConfigRoleMappings) Roles(roleMaps []string) []string {
	if len(roleMaps) == 0 {
		roleMaps = []string{"default"}
	}
	var mappedRoles []string
	for _, roleMap := range roleMaps {
		if roles, exists := crm[roleMap]; exists {
			mappedRoles = append(mappedRoles, roles...)
		}
	}
	return mappedRoles
}

// Deprecated: ArgoCD specific code will be removed from the operator
type ConfigArgoPermissions struct {
	// Deprecated: ArgoCD specific code will be removed from the operator
	// The optional default policy which is set in the ArgoCD instance
	// +kubebuilder:validation:Optional
	DefaultPolicy string `json:"default_policy,omitempty"`

	// Deprecated: ArgoCD specific code will be removed from the operator
	// The name of the ArgoCD instance to apply ArgoPermissions to
	// +kubebuilder:validation:Required
	ResourceName string `json:"resource_name"`

	// Deprecated: ArgoCD specific code will be removed from the operator
	// The name of the role to add to Groups set in ArgoPermissions
	// +kubebuilder:validation:Required
	Role string `json:"role"`

	// Deprecated: ArgoCD specific code will be removed from the operator
	// The header value to set in ArgoPermissions
	// +kubebuilder:validation:Required
	Header string `json:"header"`
}

// Deprecated: ArgoCD specific code will be removed from the operator
func (ap ConfigArgoPermissions) FromGroups(groups []string) string {
	permissions := []string{
		strings.TrimSpace(ap.Header),
	}
	for _, group := range groups {
		permissions = append(permissions, fmt.Sprintf("g, %s, role:%s", group, ap.Role))
	}
	return strings.Join(permissions, "\n")
}

type ConfigLdap struct {
	// LDAP server hostname
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// LDAP server port
	// +kubebuilder:validation:Required
	Port int32 `json:"port"`
}

type ConfigCapabilities map[string]ConfigCapability

type ConfigCapability struct {
	// Name of the ArgoCD ApplicationSet which manages this capability
	// +kubebuilder:validation:Required
	AppSet string `json:"applicationset"`

	// Quota settings for this capability
	// +kubebuilder:validation:Required
	QuotaSettings ConfigQuotaSettings `json:"quotas"`

	// Extra permissions set for this capability
	// +kubebuilder:validation:Required
	ExtraPermissions ConfigCapPerm `json:"extra_permissions"`

	// Default permissions set for this capability
	// +kubebuilder:validation:Required
	DefaultPermissions ConfigCapPerm `json:"default_permissions"`
}

type ConfigQuotaSettings struct {
	// Is this a clusterwide quota or not
	// +kubebuilder:validation:Required
	Clusterwide bool `json:"clusterwide"`

	// The ratio of the requested quota which will be applied to the total quota
	// +kubebuilder:validation:Required
	Ratio int64 `json:"ratio"`

	// The default quota which the enabled capability gets
	// +kubebuilder:validation:Required
	DefQuota map[corev1.ResourceName]resourcev1.Quantity `json:"defaults"`

	// The minimum quota which the enabled capability gets
	// +kubebuilder:validation:Required
	MinQuotas map[corev1.ResourceName]resourcev1.Quantity `json:"min"`

	// The maximum quota which the capability gets
	// +kubebuilder:validation:Required
	MaxQuotas map[corev1.ResourceName]resourcev1.Quantity `json:"max"`
}

// This is a insoudeout representation of ConfigCapPerm, closer to rb representation
type ConfigRolesSas map[string]map[string]bool

func (crs ConfigRolesSas) Merge(other ConfigRolesSas) ConfigRolesSas {
	var role map[string]bool
	var exists bool
	for rolename, sas := range other {
		if role, exists = crs[rolename]; !exists {
			role = make(map[string]bool)
		}
		for sa, add := range sas {
			role[sa] = add
		}
		crs[rolename] = role
	}
	return crs
}

type ConfigCapPerm map[string][]string

func (ccp ConfigCapPerm) AsConfigRolesSas(add bool) ConfigRolesSas {
	crs := make(ConfigRolesSas)
	for sa, roles := range ccp {
		for _, role := range roles {
			if cr, exists := crs[role]; exists {
				cr[sa] = add
				crs[role] = cr
			} else {
				cr := make(map[string]bool)
				cr[sa] = add
				crs[role] = cr
			}
		}
	}
	return crs
}

func (ccp ConfigCapPerm) Roles() []string {
	var roles []string
	for _, rolenames := range ccp {
		roles = append(roles, rolenames...)
	}
	return roles
}

func (ccp ConfigCapPerm) ServiceAccounts() []string {
	var sas []string
	for sa := range ccp {
		sas = append(sas, sa)
	}
	return sas
}

func (config PaasConfigSpec) CapabilityK8sName(capability string) (as types.NamespacedName) {
	as.Namespace = config.AppSetNamespace
	if cap, exists := config.Capabilities[capability]; exists {
		as.Name = cap.AppSet
		as.Namespace = config.AppSetNamespace
	} else {
		as.Name = fmt.Sprintf("paas-%s", capability)
		as.Namespace = config.AppSetNamespace
	}
	return as
}

type PaasConfigStatus struct {
	// Conditions of this resource
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// PaasConfigList contains a list of PaasConfig
type PaasConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PaasConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PaasConfig{}, &PaasConfigList{})
}

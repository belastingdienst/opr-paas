/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

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

func (p PaasConfig) GetConditions() []metav1.Condition {
	return p.Status.Conditions
}

type PaasConfigSpec struct {
	// DecryptKeysSecret is a reference to the secret containing the DecryptKeys
	// +kubebuilder:validation:Required
	DecryptKeysSecret NamespacedName `json:"decryptKeySecret"`

	// Enable debug information generation or not
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	Debug bool `json:"debug"`

	// A map with zero or more ConfigCapability
	// +kubebuilder:validation:Optional
	Capabilities ConfigCapabilities `json:"capabilities"`

	// Deprecated: GroupSyncList code will be removed from the operator to make it more generic
	// A reference to a configmap containing a groupsynclist of LDAP groups to be synced using LDAP sync
	// +kubebuilder:validation:Required
	GroupSyncList NamespacedName `json:"groupsynclist"`

	// Deprecated: GroupSyncListKey code will be removed from the operator to make it more generic
	// A key in the configures GroupSyncList which will contain the LDAP groups to be synced using LDAP sync
	// +kubebuilder:default:=groupsynclist.txt
	// +kubebuilder:validation:Optional
	GroupSyncListKey string `json:"groupsynclist_key"`

	// LDAP configuration for the operator to add to Groups
	// +kubebuilder:validation:Optional
	LDAP ConfigLdap `json:"ldap"`

	// Deprecated: ArgoCD specific code will be removed from the operator
	// Permissions to set for ArgoCD instance
	// +kubebuilder:validation:Optional
	ArgoPermissions ConfigArgoPermissions `json:"argopermissions"`

	// Namespace in which a clusterwide ArgoCD can be found for managing capabilities and appProjects
	// Deprecated: ArgoCD specific code will be removed from the operator
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	ClusterWideArgoCDNamespace string `json:"clusterwide_argocd_namespace"`

	// Label which is added to clusterquotas
	// +kubebuilder:default:=clusterquotagroup
	// +kubebuilder:validation:Optional
	QuotaLabel string `json:"quota_label"`

	// Name of the label used to define who is the contact for this resource
	// +kubebuilder:default:=requestor
	// +kubebuilder:validation:Optional
	RequestorLabel string `json:"requestor_label"`

	// Name of the label used to define by whom the resource is managed.
	// +kubebuilder:default:=argocd.argoproj.io/managed-by
	// +kubebuilder:validation:Optional
	ManagedByLabel string `json:"managed_by_label"`

	// Deprecated: ArgoCD specific code will be removed from the operator
	// Name of an ApplicationSet to be set as ignored in the ArgoCD bootstrap Application
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	ExcludeAppSetName string `json:"exclude_appset_name"`

	// Grant permissions to all groups according to config in configmap and role selected per group in paas.
	// +kubebuilder:validation:Optional
	RoleMappings ConfigRoleMappings `json:"rolemappings"`
}

type NamespacedName struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
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
	DefaultPolicy string `json:"default_policy"`

	// Deprecated: ArgoCD specific code will be removed from the operator
	// The name of the ArgoCD instance to apply ArgoPermissions to
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	ResourceName string `json:"resource_name"`

	// Deprecated: ArgoCD specific code will be removed from the operator
	// The name of the role to add to Groups set in ArgoPermissions
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Role string `json:"role"`

	// Deprecated: ArgoCD specific code will be removed from the operator
	// The header value to set in ArgoPermissions
	// +kubebuilder:validation:MinLength=1
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
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// LDAP server port
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	Port int32 `json:"port"`
}

type ConfigCapabilities map[string]ConfigCapability

type ConfigCapability struct {
	// Name of the ArgoCD ApplicationSet which manages this capability
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	AppSet string `json:"applicationset"`

	// Quota settings for this capability
	// +kubebuilder:validation:Required
	QuotaSettings ConfigQuotaSettings `json:"quotas"`

	// Extra permissions set for this capability
	// +kubebuilder:validation:Optional
	ExtraPermissions ConfigCapPerm `json:"extra_permissions"`

	// Default permissions set for this capability
	// +kubebuilder:validation:Optional
	DefaultPermissions ConfigCapPerm `json:"default_permissions"`

	// Settings to allow specific configuration specific to a capability
	CustomFields map[string]ConfigCustomField `json:"custom_fields,omitempty"`
}

// TODO: When we move to PaasConfig, we can probably combine Required and Default fields
// TODO: When we move to PaasConfig, we can verify Validation being a valid RE
// TODO: When we move to PaasConfig, we can verify Default meeting Validation
// TODO: When we move to PaasConfig, we can verify that Default and Required are not both set

type ConfigCustomField struct {
	// Regular expression for validating input, defaults to '', which means no validation.
	// +kubebuilder:validation:Optional
	Validation string `json:"validation"`
	// Set a default when no value is specified, defaults to ''.
	// Only applies when Required is false.
	// +kubebuilder:validation:Optional
	Default string `json:"default"`
	// Define if the value must be specified in the PaaS.
	// When set to true, and no value is set, PaasNs has error in status field, and capability is not built.
	// When set to false, and no value is set, Default is used.
	// +kubebuilder:validation:Optional
	Required bool `json:"required"`
}

type ConfigQuotaSettings struct {
	// Is this a clusterwide quota or not
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	Clusterwide bool `json:"clusterwide"`

	// The ratio of the requested quota which will be applied to the total quota
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format:=float
	// +kubebuilder:validation:Minimum:=0.0
	// +kubebuilder:validation:Maximum:=1.0
	Ratio float64 `json:"ratio"`

	// The default quota which the enabled capability gets
	// +kubebuilder:validation:Required
	DefQuota map[corev1.ResourceName]resourcev1.Quantity `json:"defaults"`

	// The minimum quota which the enabled capability gets
	// +kubebuilder:validation:Optional
	MinQuotas map[corev1.ResourceName]resourcev1.Quantity `json:"min"`

	// The maximum quota which the capability gets
	// +kubebuilder:validation:Optional
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
	as.Namespace = config.ClusterWideArgoCDNamespace
	if cap, exists := config.Capabilities[capability]; exists {
		as.Name = cap.AppSet
		as.Namespace = config.ClusterWideArgoCDNamespace
	} else {
		as.Name = fmt.Sprintf("paas-%s", capability)
		as.Namespace = config.ClusterWideArgoCDNamespace
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

// ActivePaasConfigUpdated returns a predicate to be used in watches. We are only interested in changes to the active PaasConfig.
// because we determine the active PaasConfig based on a Condition, we must use the updateFunc as the status set is done via an
// update. We explicitly don't return deletions of the PaasConfig.
func ActivePaasConfigUpdated() predicate.Predicate {
	return predicate.Funcs{
		// Trigger reconciliation only if the paasConfig has the Active PaasConfig is updated
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj := e.ObjectOld.(*PaasConfig)
			newObj := e.ObjectNew.(*PaasConfig)

			// The 'double' status check is needed because during 'creation' of the PaasConfig, the Condition is set. Once set
			// we check for specChanges.
			if meta.IsStatusConditionPresentAndEqual(newObj.Status.Conditions, TypeActivePaasConfig, metav1.ConditionTrue) {
				if !meta.IsStatusConditionPresentAndEqual(oldObj.Status.Conditions, TypeActivePaasConfig, metav1.ConditionTrue) {
					return true
				}
				return !reflect.DeepEqual(oldObj.Spec, newObj.Spec)
			}

			return false
		},

		// Disallow create events
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},

		// Disallow delete events
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},

		// Disallow generic events (e.g., external triggers)
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

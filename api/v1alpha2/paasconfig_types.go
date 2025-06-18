/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

//revive:disable:exported

package v1alpha2

import (
	"fmt"
	"reflect"

	"github.com/belastingdienst/opr-paas/v2/api"
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
	// TypeDegradedPaasConfig represents the status used when the custom resource is deleted
	// and the finalizer operations are yet to occur.
	TypeDegradedPaasConfig = "Degraded"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:conversion:hub
// +kubebuilder:resource:path=paasconfig,scope=Cluster
type PaasConfig struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PaasConfigSpec   `json:"spec,omitempty"`
	Status PaasConfigStatus `json:"status,omitempty"`
}

func (pc PaasConfig) GetConditions() []metav1.Condition {
	return pc.Status.Conditions
}

func (pc PaasConfig) GetSpec() PaasConfigSpec {
	return pc.Spec
}

func (pcs PaasConfigSpec) GetCapabilities() api.ConfigCapabilities {
	return pcs.Capabilities
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

	// Namespace in which a clusterwide ArgoCD can be found for managing capabilities and appProjects
	// Deprecated: ArgoCD specific code will be removed from the operator
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	ClusterWideArgoCDNamespace string `json:"clusterwide_argocd_namespace"`

	// Label which is added to clusterquotas
	// +kubebuilder:default:=clusterquotagroup
	// +kubebuilder:validation:Optional
	QuotaLabel string `json:"quota_label"`

	// Deprecated: RequestorLabel is replaced by go template functionality
	// Name of the label used to define who is the contact for this resource
	// +kubebuilder:default:=requestor
	// +kubebuilder:validation:Optional
	RequestorLabel string `json:"requestor_label"`

	// Deprecated: ManagedByLabel is replaced by go template functionality
	// Name of the label used to define by whom the resource is managed.
	// +kubebuilder:default:=argocd.argoproj.io/managed-by
	// +kubebuilder:validation:Optional
	ManagedByLabel string `json:"managed_by_label"`

	// Deprecated: ManagedBySuffix is replaced by go template functionality
	// once available
	// Suffix to be appended to the managed-by-label
	// +kubebuilder:default:=argocd
	// +kubebuilder:validation:Optional
	ManagedBySuffix string `json:"managed_by_suffix"`

	// Grant permissions to all groups according to config in configmap and role selected per group in paas.
	// +kubebuilder:validation:Optional
	RoleMappings ConfigRoleMappings `json:"rolemappings"`

	// Set regular expressions to have the webhooks validate the fields
	// +kubebuilder:validation:Optional
	Validations PaasConfigValidations `json:"validations"`

	// Set regular expressions to have the webhooks validate the fields
	// +kubebuilder:validation:Optional
	ResourceLabels ConfigResourceLabelConfigs `json:"resourceLabels,omitempty"`
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

// For each resource type go templating can be used to derive the labels to be set on the resource when created
type ConfigResourceLabelConfigs struct {

	// Template to describe labels for cluster quotas
	// +kubebuilder:validation:Optional
	AppSetLabels ConfigResourceLabelConfig `json:"applicationSets,omitempty"`

	// Template to describe labels for cluster quotas
	// +kubebuilder:validation:Optional
	ClusterQuotaLabels ConfigResourceLabelConfig `json:"clusterQuotas,omitempty"`

	// Template to describe labels for groups
	// +kubebuilder:validation:Optional
	GroupLabels ConfigResourceLabelConfig `json:"groups,omitempty"`

	// Template to describe labels for namespaces
	// +kubebuilder:validation:Optional
	NamespaceLabels ConfigResourceLabelConfig `json:"namespaces,omitempty"`

	// Template to describe labels for rolebindings
	// +kubebuilder:validation:Optional
	RoleBindingLabels ConfigResourceLabelConfig `json:"roleBindings,omitempty"`
}

// go templating can be used to derive the labels to be set on the resource when created
type ConfigResourceLabelConfig map[string]string

type ConfigCustomField struct {
	// Regular expression for validating input, defaults to '', which means no validation.
	// +kubebuilder:validation:Optional
	Validation string `json:"validation"`
	// Set a default when no value is specified, defaults to ''.
	// Only applies when Required is false.
	// +kubebuilder:validation:Optional
	Default string `json:"default"`
	// You can now use a go-template string to use Paas and PaasConfig variables and compile a value
	// +kubebuilder:validation:Optional
	Template string `json:"template"`
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

// TODO(hikarukin): we probably need to properly determine the namespace name,
// depends on argocd code removal
func (pcs PaasConfigSpec) CapabilityK8sName(capName string) (as types.NamespacedName) {
	if capability, exists := pcs.Capabilities[capName]; exists {
		as.Name = capability.AppSet
	} else {
		as.Name = fmt.Sprintf("paas-%s", capName)
	}
	as.Namespace = pcs.ClusterWideArgoCDNamespace

	return as
}

// revive:disable:line-length-limit

type PaasConfigStatus struct {
	// Conditions of this resource
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// revive:enable:line-length-limit

// +kubebuilder:object:root=true
// PaasConfigList contains a list of PaasConfig
type PaasConfigList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PaasConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PaasConfig{}, &PaasConfigList{})
}

// ActivePaasConfigUpdated returns a predicate to be used in watches.
// We are only interested in changes to the active PaasConfig.
// Because we determine the active PaasConfig based on a Condition,
// we must use the updateFunc as the status set is done via an update.
// We explicitly don't return deletions of the PaasConfig.
func ActivePaasConfigUpdated() predicate.Predicate {
	return predicate.Funcs{
		// Trigger reconciliation only if the paasConfig has the Active PaasConfig is updated
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj, oldOk := e.ObjectOld.(*PaasConfig)
			newObj, newOk := e.ObjectNew.(*PaasConfig)

			// If type assertion fails, return false (do not trigger reconciliation)
			if !oldOk || !newOk {
				return false
			}

			// The 'double' status check is needed because during 'creation' of the PaasConfig, the Condition is set.
			// Once set we check for specChanges.
			if meta.IsStatusConditionPresentAndEqual(
				newObj.Status.Conditions,
				TypeActivePaasConfig,
				metav1.ConditionTrue,
			) {
				if !meta.IsStatusConditionPresentAndEqual(
					oldObj.Status.Conditions,
					TypeActivePaasConfig,
					metav1.ConditionTrue,
				) {
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

// IsActive returns true if this PaasConfig is the active one.
func (pc PaasConfig) IsActive() bool {
	return meta.IsStatusConditionPresentAndEqual(
		pc.Status.Conditions,
		TypeActivePaasConfig,
		metav1.ConditionTrue,
	)
}

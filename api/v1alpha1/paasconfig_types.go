/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"
	"strings"

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

	// A reference to a configmap containing a whitelist of LDAP groups to be synced using LDAP sync
	// +kubebuilder:validation:Required
	// +kubebuilder:deprecatedversion:warning="This field is deprecated and will be removed in future versions."
	Whitelist NamespacedName `json:"whitelist"`

	// LDAP configuration for the operator to add to Groups
	// +kubebuilder:validation:Optional
	LDAP ConfigLdap `json:"ldap,omitempty"`

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

type ConfigArgoPermissions struct {
	// The optional default policy which is set in the ArgoCD instance
	// +kubebuilder:validation:Optional
	DefaultPolicy string `json:"default_policy,omitempty"`

	// The name of the ArgoCD instance to apply ArgoPermissions to
	// +kubebuilder:validation:Required
	ResourceName string `json:"resource_name"`

	// The name of the role to add to Groups set in ArgoPermissions
	// +kubebuilder:validation:Required
	Role string `json:"role"`

	// The header value to set in ArgoPermissions
	// +kubebuilder:validation:Required
	Header string `json:"header"`
}

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

// TODO use Verify in Reconciler
func (caps ConfigCapabilities) Verify() []string {
	var multierror []string
	for key, cap := range caps {
		if len(cap.QuotaSettings.DefQuota) == 0 {
			multierror = append(multierror, fmt.Sprintf("missing capabilities.%s.defaultquotas elements", key))
		}
	}
	for _, cap := range []string{"argocd", "tekton", "grafana", "sso"} {
		if _, exists := caps[cap]; !exists {
			multierror = append(multierror, fmt.Sprintf("missing capabilities.%s", cap))
		}
	}
	return multierror
}

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
	DefQuota ConfigDefaultQuotaSpec `json:"defaults"`

	// The minimum quota which the enabled capability gets
	// +kubebuilder:validation:Required
	MinQuotas ConfigDefaultQuotaSpec `json:"min"`

	// The maximum quota which the capability gets
	// +kubebuilder:validation:Required
	MaxQuotas ConfigDefaultQuotaSpec `json:"max"`
}

type ConfigDefaultQuotaSpec map[string]string

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

// const (
// 	envConfName     = "PAAS_CONFIG"
// 	defaultConfFile = "/etc/paas/config.yaml"
// )

// TODO Remove unused code, give this a place somewhere else in the operator..
// func NewConfig() (config *PaasConfig, err error) {
// 	// This only parsed as yaml, nothing else
// 	// #nosec
// 	configFile := os.Getenv(envConfName)
// 	if configFile == "" {
// 		configFile = defaultConfFile
// 	}
// 	config = &PaasConfig{}

// 	if yamlConfig, err := os.ReadFile(configFile); err != nil {
// 		return nil, err
// 	} else if err = yaml.Unmarshal(yamlConfig, config); err != nil {
// 		return nil, err
// 	} else if err = config.Verify(); err != nil {
// 		return nil, err
// 	}
// 	return config, nil
// }

// Set updates the configuration values in a thread-safe way
// func (pc *PaasConfig) Set(logLevel string, interval int) {
// 	// pc.mutex.Lock()
// 	// defer pc.mutex.Unlock()
// 	// pc.LogLevel = logLevel
// 	// pc.Interval = interval
// }

func (config PaasConfig) Verify() error {
	var multierror []string
	multierror = append(multierror, config.Spec.Capabilities.Verify()...)
	if len(multierror) > 0 {
		return fmt.Errorf("invalid config:\n%s",
			strings.Join(multierror, "\n"))
	}
	return nil
}

func (config PaasConfig) CapabilityK8sName(capability string) (as types.NamespacedName) {
	as.Namespace = config.Spec.AppSetNamespace
	if cap, exists := config.Spec.Capabilities[capability]; exists {
		as.Name = cap.AppSet
		as.Namespace = config.Spec.AppSetNamespace
	} else {
		as.Name = fmt.Sprintf("paas-%s", capability)
		as.Namespace = config.Spec.AppSetNamespace
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

	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=1
	// +kubebuilder:validation:Required
	Items []PaasConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PaasConfig{}, &PaasConfigList{})
}

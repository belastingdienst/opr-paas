/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	// TODO description
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

	// TODO description
	// +kubebuilder:validation:Optional
	// +kubebuilder:deprecatedversion:warning="This field is deprecated and will be removed in future versions."
	Whitelist types.NamespacedName `json:"whitelist,omitempty"`

	// TODO description
	// +kubebuilder:validation:Optional
	LDAP ConfigLdap `json:"ldap,omitempty"`

	// TODO description
	// +kubebuilder:validation:Optional
	ArgoPermissions ConfigArgoPermissions `json:"argopermissions,omitempty"`

	// TODO description
	// +kubebuilder:default:=argocd
	// +kubebuilder:validation:Optional
	AppSetNamespace string `json:"applicationset_namespace,omitempty"`

	// TODO description
	// +kubebuilder:default:=clusterquotagroup
	// +kubebuilder:validation:Optional
	QuotaLabel string `json:"quota_label,omitempty"`

	// Name of the label used to define who is the contact for this resource
	// +kubebuilder:default:=requestor
	// +kubebuilder:validation:Optional
	RequestorLabel string `json:"requestor_labe,omitempty"`

	// Name of the label used to define by whom the resource is managed.
	// +kubebuilder:default:=argocd.argoproj.io/managed-by
	// +kubebuilder:validation:Optional
	ManagedByLabel string `json:"managed_by_label,omitempty"`

	// TODO Description
	// +kubebuilder:validation:Required
	ExcludeAppSetName string `json:"exclude_appset_name"`

	// Grant permissions to all groups according to config in configmap and role selected per group in paas.
	// +kubebuilder:validation:Optional
	RoleMappings ConfigRoleMappings `json:"rolemappings,omitempty"`
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

/*
      rolemappings:
        edit:
          - alert-routing-edit
          - monitoring-edit
          - edit
          - neuvector
        read:
          - read
        admin:
          - admin
	  /*

/*
Feature for rolemappings:
Grant permissions to all groups according to config in configmap and role selected per group in paas.
Paas:
  groups:
    aug_cpet:
      query: >-
        CN=aug_cpet,OU=ANNA_managed,OU=AUGGroepen,OU=UID,DC=ont,DC=belastingdienst,DC=nl
      role: admin
    aug_cpet_clusteradmin:
      query: >-
        CN=aug_cpet_clusteradmin,OU=ANNA_managed,OU=AUGGroepen,OU=UID,DC=ont,DC=belastingdienst,DC=nl
      role: readonly
*/

type ConfigArgoPermissions struct {
	// TODO description
	// +kubebuilder:validation:Required
	ResourceName string `json:"resource_name"`

	// TODO description
	// +kubebuilder:validation:Required
	Role string `json:"role"`

	// TODO description
	// +kubebuilder:validation:Required
	Header string `json:"header"`

	// TODO description
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=2
	// +kubebuilder:validation:Minimum=2
	Retries uint `json:"retries,omitempty"`
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
	// TODO description
	// +kubebuilder:validation:Required
	AppSet string `json:"applicationset"`

	// TODO description
	// +kubebuilder:validation:Required
	QuotaSettings ConfigQuotaSettings `json:"quotas"`

	// TODO description
	// +kubebuilder:validation:Required
	ExtraPermissions ConfigCapPerm `json:"extra_permissions"`

	// TODO description
	// +kubebuilder:validation:Required
	DefaultPermissions ConfigCapPerm `json:"default_permissions"`
}

type ConfigQuotaSettings struct {
	// TODO description
	// +kubebuilder:validation:Required
	Clusterwide bool `json:"clusterwide"`

	// TODO description
	// +kubebuilder:validation:Required
	Ratio int64 `json:"ratio"`

	// TODO description
	// +kubebuilder:validation:Required
	DefQuota ConfigDefaultQuotaSpec `json:"defaults"`

	// TODO description
	// +kubebuilder:validation:Required
	MinQuotas ConfigDefaultQuotaSpec `json:"min"`

	// TODO description
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

const (
	envConfName     = "PAAS_CONFIG"
	defaultConfFile = "/etc/paas/config.yaml"
)

func NewConfig() (config *PaasConfig, err error) {
	// This only parsed as yaml, nothing else
	// #nosec
	configFile := os.Getenv(envConfName)
	if configFile == "" {
		configFile = defaultConfFile
	}
	config = &PaasConfig{}

	if yamlConfig, err := os.ReadFile(configFile); err != nil {
		return nil, err
	} else if err = yaml.Unmarshal(yamlConfig, config); err != nil {
		return nil, err
	} else if err = config.Verify(); err != nil {
		return nil, err
	}
	return config, nil
}

// TODO use Verfiy in Reconciler
func (config PaasConfig) Verify() error {
	var multierror []string
	if config.Spec.Whitelist.Name == "" || config.Spec.Whitelist.Namespace == "" {
		multierror = append(multierror,
			"missing whitelist.name and/or whitelist.namespace")
	}
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
	// Important: Run "make" to regenerate code after modifying this file
	Messages []string `json:"messages,omitempty"`
}

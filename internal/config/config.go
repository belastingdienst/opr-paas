/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/types"
)

type Config struct {
	filename          string
	DecryptKeyPaths   []string              `yaml:"decryptKeyPaths"`
	Debug             bool                  `yaml:"debug"`
	Capabilities      ConfigCapabilities    `yaml:"capabilities"`
	Whitelist         types.NamespacedName  `yaml:"whitelist"`
	LDAP              ConfigLdap            `yaml:"ldap"`
	ArgoPermissions   ConfigArgoPermissions `yaml:"argopermissions"`
	AppSetNamespace   string                `yaml:"applicationset_namespace"`
	QuotaLabel        string                `yaml:"quota_label"`
	RequestorLabel    string                `yaml:"requestor_label"`
	ManagedByLabel    string                `yaml:"managed_by_label"`
	ExcludeAppSetName string                `yaml:"exclude_appset_name"`
	// Grant permissions to all groups according to config in configmap and role selected per group in paas.
	RoleMappings ConfigRoleMappings `yaml:"rolemappings"`
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
	DefaultPolicy string `yaml:"default_policy,omitempty"`
	ResourceName  string `yaml:"resource_name"`
	Role          string `yaml:"role"`
	Header        string `yaml:"header"`
	Retries       uint   `yaml:"retries"`
}

func (ap ConfigArgoPermissions) Verify() []string {
	var multierror []string
	if ap.ResourceName == "" {
		multierror = append(multierror, "missing argopermissions.resource_name")
	}
	if ap.Role == "" {
		multierror = append(multierror, "missing argopermissions.role")
	}
	if ap.Header == "" {
		multierror = append(multierror, "missing argopermissions.header")
	}
	if ap.Retries < 2 {
		multierror = append(multierror, "argopermissions.retries too low")
	}
	return multierror
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
	Host string `yaml:"host"`
	Port int32  `yaml:"port"`
}

func (ldap ConfigLdap) Verify() []string {
	var multierror []string
	if ldap.Host == "" {
		multierror = append(multierror, "missing ldap.name")
	}
	if ldap.Port == 0 {
		multierror = append(multierror, "missing ldap.port")
	}
	return multierror
}

type ConfigCapabilities map[string]ConfigCapability

func (caps ConfigCapabilities) Verify() []string {
	var multierror []string
	for key, cap := range caps {
		if cap.AppSet == "" {
			multierror = append(multierror, fmt.Sprintf("missing capabilities.%s.applicationset", key))
		}
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
	AppSet             string              `yaml:"applicationset"`
	QuotaSettings      ConfigQuotaSettings `yaml:"quotas"`
	ExtraPermissions   ConfigCapPerm       `yaml:"extra_permissions"`
	DefaultPermissions ConfigCapPerm       `yaml:"default_permissions"`
}

type ConfigQuotaSettings struct {
	Clusterwide bool                   `yaml:"clusterwide"`
	Ratio       float64                `yaml:"ratio"`
	DefQuota    ConfigDefaultQuotaSpec `yaml:"defaults"`
	MinQuotas   ConfigDefaultQuotaSpec `yaml:"min"`
	MaxQuotas   ConfigDefaultQuotaSpec `yaml:"max"`
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

func NewConfig() (config *Config, err error) {
	// This only parsed as yaml, nothing else
	// #nosec
	configFile := os.Getenv(envConfName)
	if configFile == "" {
		configFile = defaultConfFile
	}
	config = &Config{
		filename: configFile,
	}

	if yamlConfig, err := os.ReadFile(configFile); err != nil {
		return nil, err
	} else if err = yaml.Unmarshal(yamlConfig, config); err != nil {
		return nil, err
	} else if err = config.Verify(); err != nil {
		return nil, err
	}
	return config, nil
}

func (config Config) Verify() error {
	var multierror []string
	if config.Whitelist.Name == "" || config.Whitelist.Namespace == "" {
		multierror = append(multierror,
			"missing whitelist.name and/or whitelist.namespace")
	}
	if config.ExcludeAppSetName == "" {
		multierror = append(multierror,
			"missing exclude_appset_name")
	}
	if len(config.DecryptKeyPaths) < 1 {
		multierror = append(multierror,
			"missing decryptKeyPaths")
	}
	if config.ManagedByLabel == "" {
		multierror = append(multierror,
			"missing managed_by_label")
	}
	if config.AppSetNamespace == "" {
		multierror = append(multierror,
			"missing applicationset_namespace")
	}
	if config.QuotaLabel == "" {
		multierror = append(multierror,
			"missing quota_label")
	}
	if config.RequestorLabel == "" {
		multierror = append(multierror,
			"missing requestor_label")
	}
	multierror = append(multierror, config.Capabilities.Verify()...)
	multierror = append(multierror, config.LDAP.Verify()...)
	multierror = append(multierror, config.ArgoPermissions.Verify()...)
	if len(multierror) > 0 {
		return fmt.Errorf("invalid config:\n%s",
			strings.Join(multierror, "\n"))
	}
	return nil
}

func (config Config) CapabilityK8sName(capability string) (as types.NamespacedName) {
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

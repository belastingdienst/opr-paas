package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/types"
)

type Config struct {
	filename        string
	DecryptKeyPath  string                `yaml:"decryptKeyPath"`
	Debug           bool                  `yaml:"debug"`
	Capabilities    ConfigCapabilities    `yaml:"capabilities"`
	Whitelist       types.NamespacedName  `yaml:"whitelist"`
	LDAP            ConfigLdap            `yaml:"ldap"`
	ArgoPermissions ConfigArgoPermissions `yaml:"argopermissions"`
	AppSetNamespace string                `yaml:"applicationset_namespace"`
	QuotaLabel      string                `yaml:"quota_label"`
	ManagedByLabel  string                `yaml:"managed_by_label"`
}

type ConfigArgoPermissions struct {
	ResourceName string `yaml:"resource_name"`
	Role         string `yaml:"role"`
	Header       string `yaml:"header"`
	Retries      uint   `yaml:"retries"`
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
		ap.Header,
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
		if len(cap.DefQuota) == 0 {
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
	AppSet   string                `yaml:"applicationset"`
	DefQuota ConfigDefaultQuotaDef `yaml:"defaultquotas"`
}

type ConfigDefaultQuotaDef map[string]string

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
	if config.DecryptKeyPath == "" {
		multierror = append(multierror,
			"missing decryptKeyPath")
	}
	if config.ManagedByLabel == "" {
		multierror = append(multierror,
			"missing managed_by_label")
	}
	multierror = append(multierror, config.Capabilities.Verify()...)
	multierror = append(multierror, config.LDAP.Verify()...)
	multierror = append(multierror, config.ArgoPermissions.Verify()...)
	multierror = append(multierror, config.Capabilities.Verify()...)
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

func (config Config) DefaultQuota(capability string) map[string]string {
	if cap, exists := config.Capabilities[capability]; exists {
		return cap.DefQuota
	}
	return nil
}

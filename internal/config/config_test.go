/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ConfigRoleMappingsRoles(t *testing.T) {
	crm := ConfigRoleMappings{}

	rolemaps := []string{}
	output := crm.Roles(rolemaps)
	assert.Nil(t, output)

	crm["Role1"] = []string{"MappedRole1", "MappedRole2"}
	crm["Role2"] = []string{"MappedRole3"}
	rolemaps = []string{"Role1", "Role2"}
	output = crm.Roles(rolemaps)
	assert.NotNil(t, output)
	assert.Equal(t, []string{"MappedRole1", "MappedRole2", "MappedRole3"}, output)
}

func Test_InvalidConfig(t *testing.T) {
	err := os.Setenv("PAAS_CONFIG", "../../test/manifests/config/paas_config_invalid.yml")
	require.NoError(t, err, "Setting env")

	_, err = NewConfig()
	assert.Error(t, err, "Reading invalid paas_config should raise an error")
}

func Test_ValidConfig(t *testing.T) {
	os.Setenv("PAAS_CONFIG", "../../test/manifests/config/paas_config.yml")
	config, err := NewConfig()
	require.NoError(t, err, "Reading valid paas_config should not raise an error")
	assert.Equal(t, "my-ldap-host", config.LDAP.Host)
	assert.Equal(t, int32(13), config.LDAP.Port)
	assert.Equal(t, "wlname", config.Whitelist.Name)
	assert.False(t, config.Debug)
	assert.Equal(t, "asns", config.AppSetNamespace)
	assert.Equal(t, "q.lbl", config.QuotaLabel)
	assert.Len(t, config.Capabilities, 4)
	assert.Equal(t, "argoas", config.Capabilities["argocd"].AppSet)
	assert.Len(t, config.Capabilities["argocd"].QuotaSettings.DefQuota, 6)
	assert.False(t, config.Capabilities["argocd"].QuotaSettings.Clusterwide)
	assert.True(t, config.Capabilities["tekton"].QuotaSettings.Clusterwide)
	assert.False(t, config.Capabilities["sso"].QuotaSettings.Clusterwide)
	assert.Len(t, config.Capabilities["argocd"].QuotaSettings.DefQuota, 6)
	assert.Equal(t, "/path/to/key", config.DecryptKeyPaths[0])
	assert.Equal(t, "argocd.argoproj.io/manby", config.ManagedByLabel)
	assert.Equal(t, "o.lbl", config.RequestorLabel)
}

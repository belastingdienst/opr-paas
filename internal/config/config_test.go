package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_InvalidConfig(t *testing.T) {
	err := os.Setenv("PAAS_CONFIG", "../../config/test/paas_config_invalid.yml")
	assert.NoError(t, err, "Setting env")

	_, err = config.NewConfig()
	assert.Error(t, err, "Reading invalid paas_config should raise an error")
}

func Test_ValidConfig(t *testing.T) {
	os.Setenv("PAAS_CONFIG", "../../config/test/paas_config.yml")
	config, err := config.NewConfig()
	assert.NoError(t, err, "Reading valid paas_config should not raise an error")
	assert.Equal(t, "my-ldap-host", config.LDAP.Host)
	assert.Equal(t, int32(13), config.LDAP.Port)
	assert.Equal(t, "wlname", config.Whitelist.Name)
	assert.Equal(t, false, config.Debug)
	assert.Equal(t, "asns", config.AppSetNamespace)
	assert.Equal(t, "q.lbl", config.QuotaLabel)
	assert.Equal(t, 4, len(config.Capabilities))
	assert.Equal(t, "argoas", config.Capabilities["argocd"].AppSet)
	assert.Equal(t, 6, len(config.Capabilities["argocd"].DefQuota))
}

/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package config

import (
	"testing"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/stretchr/testify/assert"
)

func TestGetConfigWithEmptyConfigStore(t *testing.T) {
	actual := GetConfig()

	assert.Equal(t, v1alpha2.PaasConfig{}, actual)
}

func TestGetConfig(t *testing.T) {
	cnf = PaasConfigStore{}
	SetConfig(v1alpha2.PaasConfig{
		Spec: v1alpha2.PaasConfigSpec{
			Debug: true,
		},
	})

	actual := GetConfig()

	assert.NotEmpty(t, actual)
	assert.True(t, actual.Spec.Debug)
}

func TestResetConfig(t *testing.T) {
	cnf = PaasConfigStore{}
	SetConfig(v1alpha2.PaasConfig{
		Spec: v1alpha2.PaasConfigSpec{
			Capabilities: map[string]v1alpha2.ConfigCapability{
				"x": {
					AppSet: "x",
				},
			},
		},
	})

	actual := GetConfig()
	assert.NotEmpty(t, actual)
	assert.False(t, actual.Spec.Debug)

	// Reset Config
	ResetConfig()

	// Assert get default value when no config is set
	actual = GetConfig()
	assert.Equal(t, v1alpha2.PaasConfig{}, actual)

	// Assert error when no config is set via GetConfigWithError
	_, err := GetConfigWithError()
	assert.Error(t, err, "uninitialized paasconfig")
}

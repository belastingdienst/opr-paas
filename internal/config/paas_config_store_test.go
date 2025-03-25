/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package config

import (
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestGetConfigSpecWithEmptyConfigStore(t *testing.T) {
	actual := GetConfigSpec()

	assert.Empty(t, actual)
}

func TestGetConfigSpec(t *testing.T) {
	cnf = &PaasConfigStore{
		currentConfig: v1alpha1.PaasConfig{
			Spec: v1alpha1.PaasConfigSpec{
				Debug: true,
			},
		},
	}

	actual := GetConfigSpec()

	assert.NotEmpty(t, actual)
	assert.True(t, actual.Debug)
}

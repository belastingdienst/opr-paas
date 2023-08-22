package controllers

// broken with adding ArgoPermissions capability
// Seems we need to downgrade to v.0.26.4
// (see https://github.com/operator-framework/operator-sdk/issues/6396)
// but that adds other incompatibilities.
// Maybe fix later.

import (
	"context"
	"os"
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestMain_getLogger(t *testing.T) {
	logger := getLogger(context.Background(), &v1alpha1.Paas{}, "Logger", "test")
	logger.Info("testing logging")
	logger = getLogger(context.Background(), &v1alpha1.Paas{}, "Logger", "")
	logger.Info("testing logging")
}

func TestMain_getConfig(t *testing.T) {
	os.Setenv("PAAS_CONFIG", "../config/test/paas_config.yml")
	assert.NotNil(t, getConfig(), "some-ns")
}

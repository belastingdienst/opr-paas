package controllers

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

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

func TestMain_CapName(t *testing.T) {
	os.Setenv("CAP_NAMESPACE", "some-ns")
	os.Setenv("CAP_TEST_AS_NAME", "TST_CAP")
	capName := CapabilityK8sName("test")
	assert.Equal(t, "some-ns", capName.Namespace)
	assert.Equal(t, "TST_CAP", capName.Name)
}

/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

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
	os.Setenv("PAAS_CONFIG", "../../test/manifests/config/paas_config.yml")
	assert.NotNil(t, getConfig(), "some-ns")
}

func TestMain_intersection(t *testing.T) {
	l1 := []string{"v1", "v2", "v2", "v3", "v4"}
	l2 := []string{"v2", "v2", "v3", "v5"}
	li := intersect(l1, l2)
	// Expected to have only all values that exist in list 1 and 2, only once (unique)
	lExpected := []string{"v2", "v3"}
	assert.ElementsMatch(t, li, lExpected, "result of intersection not as expected")
}

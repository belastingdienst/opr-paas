/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

import (
	"context"
	"errors"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
)

type contextKey int

const (
	contextKeyPaasConfig contextKey = iota
)

// getConfigFromContext returns the PaasConfig object from the config, using the
// contextKeyPaasConfig. If the returned value cannot be parsed to the latest
// api version PaasConfig, it returns an error.
func getConfigFromContext(ctx context.Context) (v1alpha2.PaasConfig, error) {
	myConfig, ok := ctx.Value(contextKeyPaasConfig).(v1alpha2.PaasConfig)
	if !ok {
		return v1alpha2.PaasConfig{}, errors.New("could not get config from context")
	}
	return myConfig, nil
}

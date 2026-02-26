/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type contextKey int

const (
	// ContextKeyPaasConfig is the contextKey the retrieve a PaasConfig from Context
	ContextKeyPaasConfig contextKey = iota
)

// GetConfig returns the active PaasConfig which is present in the connected kubernetes cluster
// If no (active) PaasConfig is found, it returns an error. If more than one active PaasConfig
// is found, it returns an error.
// The PaasConfig is returned as the latest API version
func GetConfig(ctx context.Context, c client.Client) (v1alpha2.PaasConfig, error) {
	var list v1alpha2.PaasConfigList
	if err := c.List(ctx, &list); err != nil {
		return v1alpha2.PaasConfig{}, fmt.Errorf("failed to retrieve PaasConfigs: %w", err)
	}
	if len(list.Items) == 0 {
		return v1alpha2.PaasConfig{}, errors.New("no PaasConfig found")
	}
	var activeConfigs v1alpha2.PaasConfigList
	for i := range list.Items {
		if list.Items[i].IsActive() {
			activeConfigs.Items = append(activeConfigs.Items, list.Items[i])
		}
	}
	if len(activeConfigs.Items) == 0 {
		return v1alpha2.PaasConfig{}, errors.New("no Active PaasConfig found")
	}
	if len(activeConfigs.Items) > 1 {
		return v1alpha2.PaasConfig{}, errors.New("multiple Active PaasConfig found")
	}
	return activeConfigs.Items[0], nil
}

// GetConfigFromContext returns the PaasConfig object from the config, using the
// config.ContextKeyPaasConfig. If the returned value cannot be parsed to the latest
// api version PaasConfig, it returns an error.
func GetConfigFromContext(ctx context.Context) (v1alpha2.PaasConfig, error) {
	myConfig, ok := ctx.Value(ContextKeyPaasConfig).(v1alpha2.PaasConfig)
	if !ok {
		return v1alpha2.PaasConfig{}, errors.New("could not get config from context")
	}
	return myConfig, nil
}

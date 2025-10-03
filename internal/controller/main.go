/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
)

type contextKey int

const (
	contextKeyPaasConfig contextKey = iota
)

func join(argv ...string) string {
	return strings.Join(argv, "-")
}

// intersect finds the intersection of 2 lists of strings
func intersect(l1 []string, l2 []string) (li []string) {
	s := map[string]bool{}
	for _, key := range l1 {
		s[key] = false
	}
	for _, key := range l2 {
		if _, exists := s[key]; exists {
			s[key] = true
		}
	}
	for key, value := range s {
		if value {
			li = append(li, key)
		}
	}
	return li
}

// Helper to merge secrets which returns a new map[string]string
func mergeSecrets(base, override map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(override))
	maps.Copy(merged, base)
	maps.Copy(merged, override)
	return merged
}

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

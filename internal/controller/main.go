/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
)

const contextKeyPaasConfig = "paasConfig"

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

// getConfigFromContext returns the PaasConfig object from the config, using the
// contextKeyPaasConfig. If the returned value cannot be parsed to the latest
// api version PaasConfig, it returns an error.
func getConfigFromContext(ctx context.Context) (v1alpha2.PaasConfig, error) {
	rawConfig := ctx.Value(contextKeyPaasConfig)
	myConfig, ok := rawConfig.(v1alpha2.PaasConfig)
	if !ok {
		return v1alpha2.PaasConfig{}, fmt.Errorf("could not get config from context")
	}
	return myConfig, nil
}

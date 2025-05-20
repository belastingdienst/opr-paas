/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

// Package config allows the current and active PaasConfig to be used all over
// the codebase.
package config

import (
	"sync"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/api/v1alpha2"
)

// PaasConfigStore is a thread-safe store for the current PaasConfig
type PaasConfigStore struct {
	mutex sync.RWMutex
	store v1alpha2.PaasConfig
}

var cnf = &PaasConfigStore{}

// GetConfig retrieves the current configuration with the latest api version
func GetConfig() v1alpha2.PaasConfig {
	cnf.mutex.RLock()
	defer cnf.mutex.RUnlock()

	return cnf.store
}

func GetConfigV1() (v1alpha1.PaasConfig, error) {
	cnf.mutex.RLock()
	defer cnf.mutex.RUnlock()

	var v1conf v1alpha1.PaasConfig
	err := (&cnf.store).ConvertTo(&v1conf)
	return v1conf, err
}

// SetConfig updates the current configuration
func SetConfig(cfg v1alpha2.PaasConfig) {
	cnf.mutex.Lock()
	defer cnf.mutex.Unlock()
	cnf.store = cfg
}

// SetConfig updates the current configuration
func SetConfigV1(cfg v1alpha1.PaasConfig) error {
	cnf.mutex.Lock()
	defer cnf.mutex.Unlock()

	return (&cnf.store).ConvertFrom(&cfg)
}

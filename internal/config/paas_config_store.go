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
)

// PaasConfigStore is a thread-safe store for the current PaasConfig
type PaasConfigStore struct {
	currentConfig v1alpha1.PaasConfig
	mutex         sync.RWMutex
}

var cnf = &PaasConfigStore{}

// GetConfig retrieVes the current configuration
func GetConfig() v1alpha1.PaasConfig {
	cnf.mutex.RLock()
	defer cnf.mutex.RUnlock()
	return cnf.currentConfig
}

// SetConfig updates the current configuration
func SetConfig(newConfig v1alpha1.PaasConfig) {
	cnf.mutex.Lock()
	defer cnf.mutex.Unlock()
	cnf.currentConfig = newConfig
}

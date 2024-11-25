/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ConfigStore is a thread-safe store for the current PaasConfig
type PaasConfigStore struct {
	currentConfig v1alpha1.PaasConfig
	mutex         sync.RWMutex
}

var (
	_cnf   = &PaasConfigStore{}
	_crypt map[string]*crypt.Crypt
)

// func initConfig() {
// 	_cnf = &v1alpha1.PaasConfig{}
// }

// GetConfig retrieves the current configuration
func GetConfig() v1alpha1.PaasConfig {
	_cnf.mutex.RLock()
	defer _cnf.mutex.RUnlock()
	return _cnf.currentConfig
}

// SetConfig updates the current configuration
func SetConfig(newConfig v1alpha1.PaasConfig) {
	_cnf.mutex.Lock()
	defer _cnf.mutex.Unlock()
	_cnf.currentConfig = newConfig
}

func getRsa(paas string) *crypt.Crypt {
	config := GetConfig()
	if _crypt == nil {
		_crypt = make(map[string]*crypt.Crypt)
	}
	if c, exists := _crypt[paas]; exists {
		return c
	} else if c, err := crypt.NewCrypt(config.Spec.DecryptKeyPaths, "", paas); err != nil {
		panic(fmt.Errorf("could not get a crypt: %w", err))
	} else {
		_crypt[paas] = c
		return c
	}
}

func getLogger(
	ctx context.Context,
	obj client.Object,
	kind string,
	name string,
) logr.Logger {
	fields := append(make([]interface{}, 0), obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), "Kind", kind)
	if name != "" {
		fields = append(fields, "Name", name)
	}

	return log.FromContext(ctx).WithValues(fields...)
}

// intersect finds the intersection of 2 lists of strings
func intersect(l1 []string, l2 []string) (li []string) {
	s := make(map[string]bool)
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

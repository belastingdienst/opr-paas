/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"sync"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// PaasConfigStore is a thread-safe store for the current PaasConfig
type PaasConfigStore struct {
	currentConfig v1alpha1.PaasConfigSpec
	mutex         sync.RWMutex
}

var (
	cnf             = &PaasConfigStore{}
	debugComponents []string
)

// GetConfig retrieves the current configuration
func GetConfig() v1alpha1.PaasConfigSpec {
	cnf.mutex.RLock()
	defer cnf.mutex.RUnlock()
	return cnf.currentConfig
}

// SetConfig updates the current configuration
func SetConfig(newConfig v1alpha1.PaasConfig) {
	cnf.mutex.Lock()
	defer cnf.mutex.Unlock()
	cnf.currentConfig = newConfig.Spec
}

// setRequestLogger derives a context with a `zerolog` logger configured for a specific controller.
// To be called once per reconciler. All functions within the reconciliation request context can access the logger with `log.Ctx()`.
func setRequestLogger(ctx context.Context, obj client.Object, scheme *runtime.Scheme, req ctrl.Request) (context.Context, *zerolog.Logger) {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		log.Err(err).Msg("failed to retrieve controller group-version-kind")

		return log.Logger.WithContext(ctx), &log.Logger
	}

	logger := log.With().
		Any("controller", gvk).
		Any("object", req.NamespacedName).
		Str("reconcileID", uuid.NewString()).
		Logger()
	logger.Info().Msg("starting reconciliation")

	return logger.WithContext(ctx), &logger
}

// SetComponentDebug configures which components will log debug messages regardless of global log level.
func SetComponentDebug(components []string) {
	debugComponents = components
}

// setLogComponent sets the component name for the logging context.
func setLogComponent(ctx context.Context, name string) context.Context {
	logger := log.Ctx(ctx)

	var found bool
	for _, c := range debugComponents {
		if c == name {
			found = true
		}
	}

	if found && logger.GetLevel() > zerolog.DebugLevel {
		ll := logger.Level(zerolog.DebugLevel)
		logger = &ll
	}

	return logger.With().Str("component", name).Logger().WithContext(ctx)
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

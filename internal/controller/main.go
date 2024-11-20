/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/crypt"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	_cnf            *config.Config
	_crypt          map[string]*crypt.Crypt
	debugComponents []string
)

func getConfig() config.Config {
	var err error
	if _cnf == nil {
		if _cnf, err = config.NewConfig(); err != nil {
			panic(fmt.Sprintf(
				"Could not read config: %s",
				err.Error()))
		}
	}
	return *_cnf
}

func getRsa(paas string) *crypt.Crypt {
	config := getConfig()
	if _crypt == nil {
		_crypt = make(map[string]*crypt.Crypt)
	}
	if c, exists := _crypt[paas]; exists {
		return c
	} else if c, err := crypt.NewCrypt(config.DecryptKeyPaths, "", paas); err != nil {
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

	return ctrllog.FromContext(ctx).WithValues(fields...)
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

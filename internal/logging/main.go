/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package logging

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	// Commandline args will use this to enable all debug logging
	staticDebug bool
	// Commandline args can use this to enable logging for a component
	staticComponents Components
	// Commandline args will use this to enable all debug logging
	dynamicDebug bool
	// Commandline args can use this to enable logging for a component
	dynamicComponents Components
)

// SetControllerLogger derives a context with a `zerolog` logger configured for a specific controller.
// To be called once per reconciler.
// All functions within the reconciliation request context can access the logger with `log.Ctx()`.
func SetControllerLogger(
	ctx context.Context,
	obj client.Object,
	scheme *runtime.Scheme,
	req ctrl.Request,
) (context.Context, *zerolog.Logger) {
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

// SetWebhookLogger derives a context with a `zerolog` logger configured for a specific object.
// To be called once per webhook validation.
// All functions within the reconciliation request context can access the logger with `log.Ctx()`.
func SetWebhookLogger(ctx context.Context, obj client.Object) (context.Context, *zerolog.Logger) {
	logger := log.With().
		Any("webhook", obj.GetObjectKind().GroupVersionKind()).
		Dict("object", zerolog.Dict().
			Str("name", obj.GetName()).
			Str("namespace", obj.GetNamespace()),
		).
		Str("requestId", uuid.NewString()).
		Logger()
	logger.Info().Msg("starting webhook validation")

	return logger.WithContext(ctx), &logger
}

// SetStaticLoggingConfig configures global debugging and component debugging from commandline argument perspective
func SetStaticLoggingConfig(debug bool, components Components) {
	staticDebug = debug
	staticComponents = components
}

// SetDynamicLoggingConfig configures global debugging and component debugging from Paas perspective
func SetDynamicLoggingConfig(debug bool, components map[Component]bool) {
	dynamicDebug = debug
	dynamicComponents = components
}

func getComponentDebugLevel(name Component) zerolog.Level {
	if enabled, exists := dynamicComponents[name]; exists {
		if enabled {
			return zerolog.DebugLevel
		}
		return zerolog.InfoLevel
	}
	if staticDebug || dynamicDebug {
		return zerolog.DebugLevel
	}
	if enabled := staticComponents[name]; enabled {
		return zerolog.DebugLevel
	}
	return zerolog.InfoLevel
}

// GetLogComponent gets the logger for a component from a context.
func GetLogComponent(ctx context.Context, name Component) (context.Context, *zerolog.Logger) {
	logger := log.Ctx(ctx)
	level := getComponentDebugLevel(name)

	if logger.GetLevel() != level {
		ll := logger.Level(level).With().Str("component", componentToString(name)).Logger()
		logger = &ll
		ctx = logger.WithContext(ctx)
	}
	logger.Debug().Msg("debugging is on")
	return ctx, logger
}

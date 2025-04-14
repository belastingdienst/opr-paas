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

var debugComponents map[string]bool

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

// SetWatcherLogger can be used to retrieve the logger for this context for this object.
// If it already exists it is just returned. If not it is created and Set before returning
func SetWatcherLogger(ctx context.Context, obj client.Object) (context.Context, *zerolog.Logger) {
	logger := log.With().
		Any("watcher", obj.GetObjectKind().GroupVersionKind()).
		Logger()
	logger.Info().Msg("starting watcher")

	return logger.WithContext(ctx), &logger
}

// ResetComponentDebug can be used to reset the map of Components that have debugging enabled
func ResetComponentDebug() {
	debugComponents = make(map[string]bool)
}

// SetComponentDebug configures which components will log debug messages regardless of global log level.
func SetComponentDebug(components []string) {
	if debugComponents == nil {
		ResetComponentDebug()
	}
	for _, component := range components {
		debugComponents[component] = true
	}
}

// GetLogComponent gets the logger for a component fro a context.
func GetLogComponent(ctx context.Context, name string) (context.Context, *zerolog.Logger) {
	logger := log.Ctx(ctx)
	level := zerolog.InfoLevel

	if _, enabled := debugComponents[name]; enabled {
		level = zerolog.DebugLevel
	}
	if logger.GetLevel() != level {
		ll := logger.Level(level)
		logger = &ll
		ctx = logger.With().Str("component", name).Logger().WithContext(ctx)
	}
	return ctx, logger
}

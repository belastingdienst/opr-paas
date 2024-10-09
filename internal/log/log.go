package log

import (
	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Get grabs the logger from the current context
func Get(ctx context.Context) logr.Logger {
	return ctrl.LoggerFrom(ctx)
}

// WithComponent derives a context with a logger namespaced by component name
func WithComponent(ctx context.Context, component string) (context.Context, logr.Logger) {
	logger := Get(ctx).WithName(component)

	return ctrl.LoggerInto(ctx, logger), logger
}

// WithAttributes derives a context with a logger with additional attributes
func WithAttributes(ctx context.Context, keysAndValues ...any) (context.Context, logr.Logger) {
	logger := Get(ctx).WithValues(keysAndValues...)

	return ctrl.LoggerInto(ctx, logger), logger
}

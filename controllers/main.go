package controllers

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var config *internal.Config

func getConfig() *internal.Config {
	var err error
	if config == nil {
		if config, err = internal.NewConfig(); err != nil {
			panic(fmt.Sprintf(
				"Could not read config: %s",
				err.Error()))
		}
	}
	return config
}

func getLogger(
	ctx context.Context,
	paas *v1alpha1.Paas,
	kind string,
	name string,
) logr.Logger {
	fields := append(make([]interface{}, 0), "Paas", paas.Name, "Kind", kind)
	if name != "" {
		fields = append(fields, "Name", name)
	}

	return log.FromContext(ctx).WithValues(fields...)
}

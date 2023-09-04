package controllers

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/rsa"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	cnf *config.Config
	r   *rsa.Rsa
)

func getConfig() config.Config {
	var err error
	if cnf == nil {
		if cnf, err = config.NewConfig(); err != nil {
			panic(fmt.Sprintf(
				"Could not read config: %s",
				err.Error()))
		}
	}
	return *cnf
}

func getRsa() *rsa.Rsa {
	if r == nil {
		r = &rsa.Rsa{}
		if _, err := r.GetPrivateKey(getConfig().DecryptKeyPath); err != nil {
			panic(fmt.Sprintf(
				"Could not read private key: %s",
				err.Error()))
		}
	}
	return r
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

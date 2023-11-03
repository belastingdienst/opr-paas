package controllers

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	_cnf   *config.Config
	_crypt map[string]*crypt.Crypt
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
	if _crypt == nil {
		_crypt = make(map[string]*crypt.Crypt)
	}
	if c, exists := _crypt[paas]; !exists {
		c = crypt.NewCrypt(getConfig().DecryptKeyPath, "", paas)
		_crypt[paas] = c
		return c
	} else {
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

/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"
	"errors"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v3/api/v1alpha1"
	cnf "github.com/belastingdienst/opr-paas/v3/internal/config"
	"github.com/belastingdienst/opr-paas/v3/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	crypts             map[string]*crypt.Crypt
	decryptPrivateKeys *crypt.PrivateKeys
)

type contextKey int

const (
	contextKeyPaasConfig contextKey = iota
)

// TODO: devotional-phoenix-97: We should refine this code and the entire crypt implementation including caching.

// resetCrypts removes all crypts and resets decryptSecretPrivateKeys
func resetCrypts() {
	crypts = map[string]*crypt.Crypt{}
	decryptPrivateKeys = nil
}

// getRsaPrivateKeys fetches secret, compares to cached private keys, resets crypts if needed, and returns keys
func getRsaPrivateKeys(ctx context.Context, _c client.Client) (*crypt.PrivateKeys, error) {
	ctx, logger := logging.GetLogComponent(ctx, logging.WebhookUtilsComponentV1)
	rsaSecret := &corev1.Secret{}
	conf, err := cnf.GetConfigV1(ctx, _c)
	if err != nil {
		return nil, err
	}
	config := conf.Spec
	namespacedName := config.DecryptKeysSecret

	err = _c.Get(ctx, types.NamespacedName{
		Name:      namespacedName.Name,
		Namespace: namespacedName.Namespace,
	}, rsaSecret)
	if err != nil {
		return nil, err
	}
	// Create new set of keys from data in secret
	keys, err := crypt.NewPrivateKeysFromSecretData(rsaSecret.Data)
	if err != nil {
		return nil, err
	}

	if decryptPrivateKeys != nil {
		if keys.Compare(*decryptPrivateKeys) {
			// It already was the same secret
			logger.Debug().Msg("reusing decrypt keys")
			return decryptPrivateKeys, nil
		}
	}

	logger.Debug().Msgf("setting (%d) new keys", len(keys))
	resetCrypts()
	decryptPrivateKeys = &keys
	return decryptPrivateKeys, nil
}

// getRsa returns a crypt.Crypt for a specified paasName
func getRsa(ctx context.Context, _c client.Client, paasName string) (*crypt.Crypt, error) {
	var c *crypt.Crypt
	if keys, err := getRsaPrivateKeys(ctx, _c); err != nil {
		return nil, err
	} else if rsa, exists := crypts[paasName]; exists {
		return rsa, nil
	} else if c, err = crypt.NewCryptFromKeys(*keys, "", paasName); err != nil {
		return nil, err
	}
	_, logger := logging.GetLogComponent(ctx, logging.WebhookUtilsComponentV1)
	logger.Debug().Msgf("creating new crypt for %s", paasName)
	crypts[paasName] = c
	return c, nil
}

// getConfigFromContext returns the PaasConfig object from the config, using the
// contextKeyPaasConfig. If the returned value cannot be parsed to a v1alpha1
// PaasConfig, it returns an error.
func getConfigFromContext(ctx context.Context) (v1alpha1.PaasConfig, error) {
	myConfig, ok := ctx.Value(contextKeyPaasConfig).(v1alpha1.PaasConfig)
	if !ok {
		return v1alpha1.PaasConfig{}, errors.New("could not get config from context")
	}
	return myConfig, nil
}

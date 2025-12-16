package controller

import (
	"context"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v4/internal/config"
	"github.com/belastingdienst/opr-paas/v4/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	// crypts contains a maps of crypt against a Paas name
	crypts             map[string]*crypt.Crypt
	decryptPrivateKeys *crypt.PrivateKeys
)

// resetCrypts removes all crypts and resets decryptSecretPrivateKeys
func resetCrypts() {
	crypts = map[string]*crypt.Crypt{}
	decryptPrivateKeys = nil
}

// getRsaPrivateKeys fetches secret, compares to cached private keys, resets crypts if needed, and returns keys
func (r *PaasReconciler) getRsaPrivateKeys(
	ctx context.Context,
) (*crypt.PrivateKeys, error) {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerSecretComponent)
	rsaSecret := &corev1.Secret{}
	cfg, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return nil, err
	}
	namespacedName := cfg.Spec.DecryptKeysSecret

	err = r.Get(ctx, types.NamespacedName{
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
func (r *PaasReconciler) getRsa(ctx context.Context, paasName string) (*crypt.Crypt, error) {
	var c *crypt.Crypt
	if keys, err := r.getRsaPrivateKeys(ctx); err != nil {
		return nil, err
	} else if rsa, exists := crypts[paasName]; exists {
		return rsa, nil
	} else if c, err = crypt.NewCryptFromKeys(*keys, "", paasName); err != nil {
		return nil, err
	}
	_, logger := logging.GetLogComponent(ctx, logging.ControllerSecretComponent)
	logger.Debug().Msgf("creating new crypt for %s", paasName)
	crypts[paasName] = c
	return c, nil
}

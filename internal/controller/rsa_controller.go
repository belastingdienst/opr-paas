package controller

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas-cli/v2/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v5/internal/config"
	"github.com/belastingdienst/opr-paas/v5/internal/logging"
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

// getCryptFunc builds a crypt and creates a func that accepts a string and returns a decrypted value (or error)
func (r *PaasReconciler) getDecryptFunc(
	ctx context.Context,
	paasName string,
) (func(string) (string, error), error) {
	keys, err := r.getRsaPrivateKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypt instance: %w", err)
	}
	rsa, err := crypt.NewCryptFromKeys(*keys, "", paasName)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypt instance: %w", err)
	}
	return func(secret string) (string, error) {
		d, decryptErr := rsa.Decrypt(secret)
		if decryptErr != nil {
			return "", decryptErr
		}
		return string(d), nil
	}, nil
}

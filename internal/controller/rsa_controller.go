package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/belastingdienst/opr-paas-cli/v2/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v5/internal/config"
	"github.com/belastingdienst/opr-paas/v5/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// getRsaPrivateKeys fetches the decryption keys secret and returns the private keys from it.
// Keys are deliberately not cached in package level state: the secret has to be read on every call anyway
// (to detect key rotation), and shared mutable state would not be safe with concurrent reconciles.
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

	logger.Debug().Msgf("using (%d) decrypt keys", len(keys))
	return &keys, nil
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
	_, logger := logging.GetLogComponent(ctx, logging.ControllerSecretComponent)
	return func(secret string) (string, error) {
		d, decryptErr := rsa.Decrypt(secret)
		if decryptErr != nil {
			// Errors from this func can end up in the Paas status (via template errors), which is readable
			// by Paas viewers. Only log the details, and return a generic error.
			logger.Debug().AnErr("error", decryptErr).Str("paas", paasName).Msg("failed to decrypt paas secret")
			return "", errSecretDecryption
		}
		return string(d), nil
	}, nil
}

// errSecretDecryption is returned when a Paas secret cannot be decrypted. It deliberately has no details.
var errSecretDecryption = errors.New("paas secret cannot be decrypted")

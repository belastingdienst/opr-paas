package controller

import (
	"context"
	"errors"

	"github.com/belastingdienst/opr-paas/v5/internal/config"
	"github.com/belastingdienst/opr-paas/v5/internal/decrypt"
	"github.com/belastingdienst/opr-paas/v5/internal/logging"
)

// getCryptFunc builds a crypt and creates a func that accepts a string and returns a decrypted value (or error)
func (r *PaasReconciler) getDecryptFunc(
	ctx context.Context,
	paasName string,
) (func(string) (string, error), error) {
	cfg, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rsa, err := decrypt.ForPaas(ctx, r.Client, cfg, paasName)
	if err != nil {
		return nil, err
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

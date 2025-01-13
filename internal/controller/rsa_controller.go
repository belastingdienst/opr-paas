package controller

import (
	"context"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	// crypts contains a maps of crypt against a Paas name
	crypts             map[string]*crypt.Crypt
	decryptPrivateKeys *crypt.CryptPrivateKeys
)

// resetCrypts removes all crypts and resets decryptSecretPrivateKeys
func resetCrypts() {
	crypts = nil
	decryptPrivateKeys = nil
}

// getOrEnsureRsaSecret ensures that secret exists creating an empty secret if needed
// and returns the body of the fetched or created secret
func (r *PaasNSReconciler) getOrEnsureRsaSecret(
	ctx context.Context,
) (keys *crypt.CryptPrivateKeys, err error) {
	logger := log.Ctx(ctx)
	// See if rsa secret exists and create if it doesn't
	rsaSecret := &corev1.Secret{}
	config := GetConfig()
	namespacedName := config.DecryptKeysSecret

	err = r.Get(ctx, types.NamespacedName{
		Name:      namespacedName.Name,
		Namespace: namespacedName.Namespace,
	}, rsaSecret)
	if err != nil && errors.IsNotFound(err) {
		logger.Debug().Msg("decrypt key secret not yet defined, creating from files")
		decryptPrivateKeysFromFiles, err := crypt.NewPrivateKeysFromFiles(config.DecryptKeyPaths)
		if err != nil {
			return nil, err
		}
		// Create the rsa secret
		rsaSecret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
			Data: decryptPrivateKeysFromFiles.AsSecretData(),
		}
		if err = r.Create(ctx, rsaSecret); err != nil {
			return nil, err
		}
		return &decryptPrivateKeysFromFiles, nil
	} else if err != nil {
		// Error that isn't due to the secret not existing
		return nil, err
	}
	// Create new set of keys from data in secret
	decryptPrivateKeysFromFiles, err := crypt.NewPrivateKeysFromSecretData(rsaSecret.Data)
	return &decryptPrivateKeysFromFiles, err
}

func (r *PaasNSReconciler) refreshRsaPrivateKeys(ctx context.Context) (keys *crypt.CryptPrivateKeys, err error) {
	// Secret changed? Yes: Reset map, get keys, update SecretGen
	// If one error occurs, all is invalid

	ctx = setLogComponent(ctx, "rolebinding")
	logger := log.Ctx(ctx)
	keys, err = r.getOrEnsureRsaSecret(ctx)
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

	decryptPrivateKeys = keys
	logger.Debug().Msgf("setting (%d) new keys", len(*keys))
	crypts = make(map[string]*crypt.Crypt)
	return
}

// getRsa returns a crypt.Crypt for a specified paasName
func (r *PaasNSReconciler) getRsa(ctx context.Context, paasName string) (*crypt.Crypt, error) {
	if keys, err := r.refreshRsaPrivateKeys(ctx); err != nil {
		return nil, err
	} else if rsa, exists := crypts[paasName]; exists {
		return rsa, nil
	} else if c, err := crypt.NewCryptFromKeys(*keys, "", paasName); err != nil {
		return nil, err
	} else {
		logger := log.Ctx(ctx)
		logger.Debug().Msgf("creating new crypt for %s", paasName)
		crypts[paasName] = c
		return c, nil
	}
}

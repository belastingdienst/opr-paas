/*
Copyright 2026, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

// Package decrypt is the one place the operator reads the decryption keys of
// Paas secrets and builds the crypt that decrypts them. The Paas controller,
// the Paas and PaasNS webhooks and the Argo CD plugin generator all use it, so
// a change of key format or algorithm is made once.
package decrypt

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas-cli/v2/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PrivateKeys reads the secret named in spec.decryptKeySecret of conf and
// returns the private keys in it. The keys are deliberately not cached: the
// secret has to be read on every call anyway (to notice key rotation), and
// shared mutable state would not be safe with concurrent reconciles.
func PrivateKeys(ctx context.Context, reader client.Reader, conf v1alpha2.PaasConfig) (crypt.PrivateKeys, error) {
	secret := &corev1.Secret{}
	if err := reader.Get(ctx, types.NamespacedName{
		Name:      conf.Spec.DecryptKeysSecret.Name,
		Namespace: conf.Spec.DecryptKeysSecret.Namespace,
	}, secret); err != nil {
		return nil, fmt.Errorf("could not retrieve decryption secret: %w", err)
	}
	keys, err := crypt.NewPrivateKeysFromSecretData(secret.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private keys: %w", err)
	}
	return keys, nil
}

// NewCrypt returns a crypt that decrypts the secrets of the Paas named paasName
// with keys.
func NewCrypt(keys crypt.PrivateKeys, paasName string) (*crypt.Crypt, error) {
	c, err := crypt.NewCryptFromKeys(keys, "", paasName)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypt instance: %w", err)
	}
	return c, nil
}

// ForPaas reads the keys and returns a crypt for the Paas named paasName.
func ForPaas(
	ctx context.Context,
	reader client.Reader,
	conf v1alpha2.PaasConfig,
	paasName string,
) (*crypt.Crypt, error) {
	keys, err := PrivateKeys(ctx, reader, conf)
	if err != nil {
		return nil, err
	}
	return NewCrypt(keys, paasName)
}

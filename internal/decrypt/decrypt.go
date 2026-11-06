/*
Copyright 2026, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

// Package decrypt is the one place where the operator turns Paas secrets into plaintext. The Paas controller, the
// Paas and PaasNS webhooks and the Argo CD plugin generator all use it, so key loading, caching and error handling
// are the same everywhere.
package decrypt

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"

	"github.com/belastingdienst/opr-paas-cli/v2/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// parsePrivateKeys parses the data of the keys secret. A variable, so tests can count the calls.
var parsePrivateKeys = crypt.NewPrivateKeysFromSecretData

// keyCache holds the private keys parsed from the keys secret, with a digest of the secret data they were parsed from.
type keyCache struct {
	mu     sync.Mutex
	secret types.NamespacedName
	digest [sha256.Size]byte
	keys   crypt.PrivateKeys
}

// cache is shared by every caller in the process. The keys secret is parsed once per version of its data.
var cache keyCache

// PrivateKeys returns the private keys from the keys secret that the PaasConfig names in spec.decryptKeySecret.
//
// The secret is read on every call (through the client's cache), but parsed only when its data has changed since the
// last call, because parsing RSA private keys is expensive. The cache is keyed on the secret's name and a digest of
// its data, so a recreated or updated secret is always parsed again.
func PrivateKeys(ctx context.Context, reader client.Reader, conf v1alpha2.PaasConfig) (crypt.PrivateKeys, error) {
	name := types.NamespacedName{
		Name:      conf.Spec.DecryptKeysSecret.Name,
		Namespace: conf.Spec.DecryptKeysSecret.Namespace,
	}
	secret := &corev1.Secret{}
	if err := reader.Get(ctx, name, secret); err != nil {
		return nil, fmt.Errorf("could not retrieve decryption secret: %w", err)
	}
	digest := dataDigest(secret.Data)

	cache.mu.Lock()
	defer cache.mu.Unlock()
	logger := log.Ctx(ctx)
	if cache.keys != nil && cache.secret == name && cache.digest == digest {
		logger.Debug().Msg("reusing decrypt keys")
		return cache.keys, nil
	}
	keys, err := parsePrivateKeys(secret.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private keys: %w", err)
	}
	logger.Debug().Msgf("setting (%d) new decrypt keys", len(keys))
	cache.secret, cache.digest, cache.keys = name, digest, keys
	return keys, nil
}

// Decrypter decrypts the secrets of one Paas.
type Decrypter struct {
	crypt *crypt.Crypt
}

// NewDecrypter returns a Decrypter for the secrets of the Paas named paasName, using the keys from the keys secret
// that the PaasConfig names.
func NewDecrypter(
	ctx context.Context,
	reader client.Reader,
	conf v1alpha2.PaasConfig,
	paasName string,
) (*Decrypter, error) {
	keys, err := PrivateKeys(ctx, reader, conf)
	if err != nil {
		return nil, err
	}
	return NewDecrypterFromKeys(keys, paasName)
}

// NewDecrypterFromKeys returns a Decrypter for the secrets of the Paas named paasName, using keys that the caller
// already loaded with PrivateKeys (for example once for many Paases).
func NewDecrypterFromKeys(keys crypt.PrivateKeys, paasName string) (*Decrypter, error) {
	c, err := crypt.NewCryptFromKeys(keys, "", paasName)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypt instance: %w", err)
	}
	return &Decrypter{crypt: c}, nil
}

// Decrypt returns the plaintext of an encrypted Paas secret. Its signature matches the decryptPaasSecret template
// function.
func (d *Decrypter) Decrypt(secret string) (string, error) {
	plain, err := d.crypt.Decrypt(secret)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// dataDigest is a digest of the secret data, independent of map order.
func dataDigest(data map[string][]byte) [sha256.Size]byte {
	names := make([]string, 0, len(data))
	for name := range data {
		names = append(names, name)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		// Length-prefix both parts, so ("ab", "c") and ("a", "bc") differ.
		fmt.Fprintf(h, "%d:%s%d:", len(name), name, len(data[name]))
		h.Write(data[name])
	}
	var digest [sha256.Size]byte
	copy(digest[:], h.Sum(nil))
	return digest
}

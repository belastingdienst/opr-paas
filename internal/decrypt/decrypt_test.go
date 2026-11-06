/*
Copyright 2026, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package decrypt

import (
	"context"
	"sync"
	"testing"

	"github.com/belastingdienst/opr-paas-cli/v2/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	secretName      = "paas-decrypt-keys"
	secretNamespace = "paas-system"
	paasName        = "my-paas"
	concurrentCalls = 20
)

var testConfig = v1alpha2.PaasConfig{
	Spec: v1alpha2.PaasConfigSpec{
		DecryptKeysSecret: v1alpha2.NamespacedName{Name: secretName, Namespace: secretNamespace},
	},
}

// resetCache empties the shared cache and counts the parses until the test ends.
func resetCache(t *testing.T) *int {
	t.Helper()
	cache = keyCache{}
	parses := 0
	var mu sync.Mutex
	original := parsePrivateKeys
	parsePrivateKeys = func(data map[string][]byte) (crypt.PrivateKeys, error) {
		mu.Lock()
		parses++
		mu.Unlock()
		return original(data)
	}
	t.Cleanup(func() {
		parsePrivateKeys = original
		cache = keyCache{}
	})
	return &parses
}

func newKeys(t *testing.T) crypt.PrivateKeys {
	t.Helper()
	pk, err := crypt.GeneratePrivateKey()
	require.NoError(t, err)
	return crypt.PrivateKeys{pk.GetID(): pk}
}

func keysSecret(keys crypt.PrivateKeys) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace},
		Data:       keys.AsSecretData(),
	}
}

func newClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func encrypt(t *testing.T, keys crypt.PrivateKeys, encryptionContext string, plain string) string {
	t.Helper()
	pub, err := keys.PublicKey()
	require.NoError(t, err)
	c, err := crypt.NewCryptFromKeys(keys, pub, encryptionContext)
	require.NoError(t, err)
	encrypted, err := c.Encrypt([]byte(plain))
	require.NoError(t, err)
	return encrypted
}

func TestDecrypterDecryptsASecretOfItsPaas(t *testing.T) {
	resetCache(t)
	keys := newKeys(t)
	cl := newClient(t, keysSecret(keys))

	d, err := NewDecrypter(context.Background(), cl, testConfig, paasName)
	require.NoError(t, err)
	plain, err := d.Decrypt(encrypt(t, keys, paasName, "s3cr3t"))
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", plain)

	// The Paas name is the encryption context: another Paas's secret does not decrypt.
	_, err = d.Decrypt(encrypt(t, keys, "other-paas", "s3cr3t"))
	assert.Error(t, err)
}

func TestTheKeysSecretIsParsedOncePerVersion(t *testing.T) {
	parses := resetCache(t)
	keys := newKeys(t)
	cl := newClient(t, keysSecret(keys))
	ctx := context.Background()

	for range 3 {
		_, err := PrivateKeys(ctx, cl, testConfig)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, *parses, "an unchanged secret is parsed once")
}

func TestAChangedKeysSecretIsParsedAgain(t *testing.T) {
	parses := resetCache(t)
	oldKeys, newKeysSet := newKeys(t), newKeys(t)
	secret := keysSecret(oldKeys)
	cl := newClient(t, secret)
	ctx := context.Background()

	_, err := PrivateKeys(ctx, cl, testConfig)
	require.NoError(t, err)

	secret.Data = newKeysSet.AsSecretData()
	require.NoError(t, cl.Update(ctx, secret))
	d, err := NewDecrypter(ctx, cl, testConfig, paasName)
	require.NoError(t, err)

	assert.Equal(t, 2, *parses)
	plain, err := d.Decrypt(encrypt(t, newKeysSet, paasName, "rotated"))
	require.NoError(t, err)
	assert.Equal(t, "rotated", plain)
}

func TestAnUnparsableKeysSecretIsAnErrorAndIsNotCached(t *testing.T) {
	resetCache(t)
	keys := newKeys(t)
	broken := keysSecret(keys)
	broken.Data = map[string][]byte{"key": []byte("not a private key")}
	cl := newClient(t, broken)
	ctx := context.Background()

	_, err := NewDecrypter(ctx, cl, testConfig, paasName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse private keys")

	broken.Data = keys.AsSecretData()
	require.NoError(t, cl.Update(ctx, broken))
	_, err = NewDecrypter(ctx, cl, testConfig, paasName)
	assert.NoError(t, err, "after the secret is fixed, it is parsed and used")
}

func TestAMissingKeysSecretIsAnError(t *testing.T) {
	resetCache(t)
	_, err := NewDecrypter(context.Background(), newClient(t), testConfig, paasName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not retrieve decryption secret")
}

func TestASecretRecreatedWithOtherKeysIsNotServedFromTheCache(t *testing.T) {
	resetCache(t)
	oldKeys, newKeysSet := newKeys(t), newKeys(t)
	ctx := context.Background()

	_, err := PrivateKeys(ctx, newClient(t, keysSecret(oldKeys)), testConfig)
	require.NoError(t, err)
	// A new client, as after a restart of the API server's view: same name, other data.
	d, err := NewDecrypter(ctx, newClient(t, keysSecret(newKeysSet)), testConfig, paasName)
	require.NoError(t, err)
	plain, err := d.Decrypt(encrypt(t, newKeysSet, paasName, "fresh"))
	require.NoError(t, err)
	assert.Equal(t, "fresh", plain)
}

func TestConcurrentCallersShareOneParse(t *testing.T) {
	parses := resetCache(t)
	keys := newKeys(t)
	cl := newClient(t, keysSecret(keys))
	encrypted := encrypt(t, keys, paasName, "shared")

	var wg sync.WaitGroup
	errs := make(chan error, concurrentCalls)
	for range concurrentCalls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, err := NewDecrypter(context.Background(), cl, testConfig, paasName)
			if err == nil {
				_, err = d.Decrypt(encrypted)
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		assert.NoError(t, err)
	}
	assert.Equal(t, 1, *parses)
}

func TestDataDigestIgnoresMapOrderButNotContent(t *testing.T) {
	a := map[string][]byte{"one": []byte("1"), "two": []byte("2")}
	b := map[string][]byte{"two": []byte("2"), "one": []byte("1")}
	c := map[string][]byte{"on": []byte("e1"), "two": []byte("2")}
	assert.Equal(t, dataDigest(a), dataDigest(b))
	assert.NotEqual(t, dataDigest(a), dataDigest(c))
}

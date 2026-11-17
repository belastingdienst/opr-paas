/*
Copyright 2026, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package decrypt

import (
	"context"
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
)

var testConfig = v1alpha2.PaasConfig{
	Spec: v1alpha2.PaasConfigSpec{
		DecryptKeysSecret: v1alpha2.NamespacedName{Name: secretName, Namespace: secretNamespace},
	},
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

func TestForPaasDecryptsASecretOfItsPaas(t *testing.T) {
	keys := newKeys(t)
	c, err := ForPaas(context.Background(), newClient(t, keysSecret(keys)), testConfig, paasName)
	require.NoError(t, err)

	plain, err := c.Decrypt(encrypt(t, keys, paasName, "s3cr3t"))
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", string(plain))

	// The Paas name is the encryption context: another Paas's secret does not decrypt.
	_, err = c.Decrypt(encrypt(t, keys, "other-paas", "s3cr3t"))
	assert.Error(t, err)
}

func TestAChangedKeysSecretIsReadOnTheNextCall(t *testing.T) {
	oldKeys, rotated := newKeys(t), newKeys(t)
	secret := keysSecret(oldKeys)
	cl := newClient(t, secret)
	ctx := context.Background()

	_, err := ForPaas(ctx, cl, testConfig, paasName)
	require.NoError(t, err)

	secret.Data = rotated.AsSecretData()
	require.NoError(t, cl.Update(ctx, secret))
	c, err := ForPaas(ctx, cl, testConfig, paasName)
	require.NoError(t, err)

	plain, err := c.Decrypt(encrypt(t, rotated, paasName, "rotated"))
	require.NoError(t, err)
	assert.Equal(t, "rotated", string(plain))
}

func TestAMissingKeysSecretIsAnError(t *testing.T) {
	_, err := ForPaas(context.Background(), newClient(t), testConfig, paasName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not retrieve decryption secret")
}

func TestAnUnparsableKeysSecretIsAnError(t *testing.T) {
	broken := keysSecret(newKeys(t))
	broken.Data = map[string][]byte{"key": []byte("not a private key")}

	_, err := ForPaas(context.Background(), newClient(t, broken), testConfig, paasName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse private keys")
}

/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package crypt

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RsaGenerate(t *testing.T) {
	priv, err := os.CreateTemp("", "private")
	require.NoError(t, err, "Creating tempfile for private key")
	defer os.Remove(priv.Name()) // clean up

	pub, err := os.CreateTemp("", "public")
	require.NoError(t, err, "Creating tempfile for public key")
	defer os.Remove(pub.Name()) // clean up

	c, err := NewGeneratedCrypt(priv.Name(), pub.Name())
	require.NoError(t, err, "Crypt object created")
	assert.NotNil(t, c, "Crypt object is not nil")
}

func Test_Rsa(t *testing.T) {
	c, err := NewCrypt(
		[]string{"../../testdata/private.rsa.key"},
		"../../testdata/public.rsa.key",
		"",
	)
	require.NoError(t, err, "Getting New Crypt")

	original := "CPET_is_the_best"

	encrypted, err := c.EncryptRsa([]byte(original))
	require.NoError(t, err, "Encrypting data")

	decrypted, err := c.DecryptRsa(encrypted)
	require.NoError(t, err, "Decrypting data")
	assert.Equal(t, string(decrypted), string(original))
}

/*
func Test_Aes(t *testing.T) {
	original := []byte("Dit is een test")
	key := "Dit is de key"
	wrong_key := "Dit is de verkeerde key"

	c := NewCrypt("", "", key)
	encrypted, err := c.EncryptAes(original)
	assert.NoError(t, err, "Encrypting")
	assert.Greater(t, len(encrypted), 0)

	decrypted, err := c.DecryptAes(encrypted)
	assert.NoError(t, err, "Decrypting")
	assert.Equal(t, string(original), string(decrypted))

	wrong_key_decryption, err := NewCrypt("", "", wrong_key).DecryptAes(encrypted)
	assert.Error(t, err, "Decrypting with wrong key")
	assert.Nil(t, wrong_key_decryption, "Encrypting with wrong key gives no data")
}
*/

func Test_Crypt(t *testing.T) {
	original := "Dit is een test"
	c, err := NewCrypt(
		[]string{"../../testdata/private.rsa.key"},
		"../../testdata/public.rsa.key",
		"Dit is de key",
	)
	require.NoError(t, err, "Getting New Crypt")
	encrypted, err := c.Encrypt([]byte(original))
	require.NoError(t, err, "Encrypting")
	assert.Greater(t, len(encrypted), 100)

	decrypted, err := c.Decrypt(encrypted)
	require.NoError(t, err, "Decrypting")
	assert.Equal(t, original, string(decrypted))
}

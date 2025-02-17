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

func TestRsaGenerate(t *testing.T) {
	context := "context"
	priv, err := os.CreateTemp("", "private")
	require.NoError(t, err, "Creating tempfile for private key")
	defer os.Remove(priv.Name()) // clean up

	pub, err := os.CreateTemp("", "public")
	require.NoError(t, err, "Creating tempfile for public key")
	defer os.Remove(pub.Name()) // clean up

	c, err := NewGeneratedCrypt(priv.Name(), pub.Name(), context)
	require.NoError(t, err, "Crypt object created")
	assert.NotNil(t, c, "Crypt object is not nil")
}

func TestRsa(t *testing.T) {
	context := "context"

	// generate private/public keys
	priv, err := os.CreateTemp("", "private")
	require.NoError(t, err, "Creating tempfile for private key")
	defer os.Remove(priv.Name()) // clean up

	pub, err := os.CreateTemp("", "public")
	require.NoError(t, err, "Creating tempfile for public key")
	defer os.Remove(pub.Name()) // clean up

	c, err := NewGeneratedCrypt(priv.Name(), pub.Name(), context)

	require.NoError(t, err, "Getting New Crypt")

	original := "CPET_is_the_best"

	encrypted, err := c.EncryptRsa([]byte(original))
	require.NoError(t, err, "Encrypting data")

	decrypted, err := c.DecryptRsa(encrypted)
	require.NoError(t, err, "Decrypting data")
	assert.Equal(t, string(decrypted), string(original))
}

func TestCrypt(t *testing.T) {
	const (
		minimalEncryptedLength = 100
	)
	var (
		original = "Dit is een test"
		context1 = "context1"
		context2 = "context2"
	)

	// generate private/public keys
	priv, err := os.CreateTemp("", "private")
	require.NoError(t, err, "Creating tempfile for private key")
	defer os.Remove(priv.Name()) // clean up

	pub, err := os.CreateTemp("", "public")
	require.NoError(t, err, "Creating tempfile for public key")
	defer os.Remove(pub.Name()) // clean up

	c, err := NewGeneratedCrypt(priv.Name(), pub.Name(), context1)
	require.NoError(t, err, "Getting New Crypt")

	encrypted, err := c.Encrypt([]byte(original))
	require.NoError(t, err, "Encrypting")
	assert.Greater(t, len(encrypted), minimalEncryptedLength)

	decrypted, err := c.Decrypt(encrypted)
	require.NoError(t, err, "Decrypting")
	assert.Equal(t, original, string(decrypted))

	c.encryptionContext = []byte(context2)
	_, err = c.Decrypt(encrypted)
	require.Error(t, err, "Decrypting with other context")
	encrypted, err = c.Encrypt([]byte(original))
	require.NoError(t, err, "Encrypting with other context should succeed")
	decrypted, err = c.Decrypt(encrypted)
	require.NoError(t, err, "Decrypting with other context")
	assert.Equal(t, original, string(decrypted))
}

package crypt

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_RsaGenerate(t *testing.T) {
	priv, err := os.CreateTemp("", "private")
	assert.NoError(t, err, "Creating tempfile for private key")
	defer os.Remove(priv.Name()) // clean up

	pub, err := os.CreateTemp("", "public")
	assert.NoError(t, err, "Creating tempfile for public key")
	defer os.Remove(pub.Name()) // clean up

	c, err := NewCrypt(priv.Name(), pub.Name(), "").Generate()
	assert.NoError(t, err, "Crypt object created")
	assert.NotNil(t, c, "Crypt object is not nil")
}

func Test_Rsa(t *testing.T) {
	c := NewCrypt(
		"../../testdata/private.rsa.key",
		"../../testdata/public.rsa.key",
		"",
	)

	original := "CPET_is_the_best"

	encrypted, err := c.EncryptRsa([]byte(original))
	assert.NoError(t, err, "Encrypting data")

	decrypted, err := c.DecryptRsa(encrypted)
	assert.NoError(t, err, "Decrypting data")
	assert.Equal(t, string(decrypted), string(original))
}

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

func Test_Crypt(t *testing.T) {
	original := "Dit is een test"
	c := NewCrypt(
		"../../testdata/private.rsa.key",
		"../../testdata/public.rsa.key",
		"Dit is de key",
	)
	encrypted, err := c.Encrypt(original)
	assert.NoError(t, err, "Encrypting")
	assert.Greater(t, len(encrypted), 100)

	decrypted, err := c.Decrypt(encrypted)
	assert.NoError(t, err, "Decrypting")
	assert.Equal(t, original, decrypted)
}

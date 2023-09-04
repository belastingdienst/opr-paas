package aes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_hashedKey(t *testing.T) {
	key := []byte("Dit is een key")
	hashed := hashedKey(key)
	assert.NotEqual(t, string(key), string(hashed), "hashing should change the key")
	assert.Equal(t, len(hashed), 16, "hashing should give a key of 16 bytes")
}

func Test_Encrypt(t *testing.T) {
	original := []byte("Dit is een test")
	key := []byte("Dit is de key")
	key2 := []byte("Dit is de verkeerde key")

	encrypted, err := Encrypt(original, key)
	assert.NoError(t, err, "Encrypting")
	assert.Greater(t, len(encrypted), 0)

	decrypted, err := Decrypt(encrypted, key)
	assert.NoError(t, err, "Decrypting")
	assert.Equal(t, string(original), string(decrypted))

	wrong_key_decryption, err := Decrypt(encrypted, key2)
	assert.Error(t, err, "Decrypting with wrong key")
	assert.Nil(t, wrong_key_decryption, "Encrypting with wrong key gives no data")
}

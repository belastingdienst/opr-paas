package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"io"
)

func hashedKey(key []byte) []byte {
	hasher := sha1.New()
	hasher.Write(key)
	return hasher.Sum(nil)[:16]
}

func Encrypt(decrypted []byte, key []byte) ([]byte, error) {
	if c, err := aes.NewCipher(hashedKey(key)); err != nil {
		return nil, fmt.Errorf("could not create new AES cypher: %e", err)
	} else if gcm, err := cipher.NewGCM(c); err != nil {
		// gcm or Galois/Counter Mode, is a mode of operation
		// for symmetric key cryptographic block ciphers
		// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
		return nil, fmt.Errorf("could not create new GCM: %e", err)
	} else {
		// creates a new byte array the size of the nonce
		// which must be passed to Seal
		nonce := make([]byte, gcm.NonceSize())

		// populates our nonce with a cryptographically secure
		// random sequence
		if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
			fmt.Println(err)
		}

		// here we encrypt our text using the Seal function
		// Seal encrypts and authenticates plaintext, authenticates the
		// additional data and appends the result to dst, returning the updated
		// slice. The nonce must be NonceSize() bytes long and unique for all
		// time, for a given key.
		return gcm.Seal(nonce, nonce, decrypted, nil), nil
	}
}

func Decrypt(encrypted []byte, key []byte) ([]byte, error) {

	if c, err := aes.NewCipher(hashedKey(key)); err != nil {
		return nil, fmt.Errorf("could not create new AES cypher: %e", err)
	} else if gcm, err := cipher.NewGCM(c); err != nil {
		return nil, fmt.Errorf("could not create new GCM: %e", err)
	} else if nonceSize := gcm.NonceSize(); len(encrypted) < nonceSize {
		return nil, fmt.Errorf("invalid encrypted data (%d smaller than nonce %d)", len(encrypted), nonceSize)
	} else {
		nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
		if plaintext, err := gcm.Open(nil, nonce, ciphertext, nil); err != nil {
			return nil, fmt.Errorf("decryption failed: %e", err)
		} else {
			return plaintext, nil
		}
	}
}

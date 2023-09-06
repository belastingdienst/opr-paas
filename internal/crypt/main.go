package crypt

import (
	"crypto/sha1"
)

func hashedKey(key []byte) []byte {
	hasher := sha1.New()
	hasher.Write(key)
	return hasher.Sum(nil)[:16]
}

/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package crypt

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

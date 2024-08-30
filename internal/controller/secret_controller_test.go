/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashData(t *testing.T) {
	testString1 := "My Wonderful Test String"
	testString2 := "Another Wonderful Test String"

	out1 := hashData(testString1)
	out2 := hashData(testString2)

	assert.Equal(t, "703fe1668c39ec0fdf3c9916d526ba4461fe10fd36bac1e2a1b708eb8a593e418eb3f92dbbd2a6e3776516b0e03743a45cfd69de6a3280afaa90f43fa1918f74", out1)
	assert.Equal(t, "d3bfd910013886fe68ffd5c5d854e7cb2a8ce2a15a48ade41505b52ce7898f63d8e6b9c84eacdec33c45f7a2812d93732b524be91286de328bbd6b72d5aee9de", out2)
}

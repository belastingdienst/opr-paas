/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_formatEndpoint(t *testing.T) {
	// test: empty endpoint
	output := formatEndpoint("")
	require.NotNil(t, output)
	assert.Equal(t, ":8080", output)

	// test: endpoint contains anything but ':'
	output = formatEndpoint("abcdef")
	require.NotNil(t, output)
	assert.Equal(t, "abcdef:8080", output)

	// test: endpoint contains ':'
	output = formatEndpoint("ABC.DEF:3000")
	require.NotNil(t, output)
	assert.Equal(t, "ABC.DEF:3000", output)

	// test: endpoint contains ':' & hostname too long
	// note: test hostname is 70 characters long
	require.Panics(
		t,
		func() { formatEndpoint("abcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghij:3000") },
		"Should panic because hostname is too long",
	)

	// test: endpoint contains ':' & hostname is invalid
	for _, ep := range []string{
		"abc.DEF:3000",
		"abc.DEF",
		"abc.DEF.nl",
		"abc",
		"abc.DEF-ghi.net:3000",
	} {
		require.NotPanics(
			t,
			func() { formatEndpoint(ep) },
			"Should NOT panic because hostname is valid (sanity check)",
		)
	}
	for ep, msg := range map[string]string{
		"abc#DEF:3000":     "contains illegal character (#)",
		".abcDEF:3000":     "contains illegal character (.)",
		"ab..cDEF:3000":    "contains illegal characters (..)",
		"-abcDEF:3000":     "starts with illegal character (-)",
		"abc.DEF-:3000":    "ends with illegal character (-)",
		"abc.DEF.666:3000": "TLD contains illegal string (666)",
		"abc.DEF_ghi:3000": "contains illegal character (_)",
	} {
		require.Panics(
			t,
			func() { formatEndpoint(ep) },
			"Should panic because hostname "+msg,
		)
	}
	assert.Panics(t, func() { formatEndpoint("abc.DEF.a:3000") }, "Should panic because hostname TLD too short (<2)")

	// test: endpoint contains ':' & portnum is empty
	output = formatEndpoint("my.valid.host:")
	assert.NotNil(t, output)
	assert.Equal(t, "my.valid.host:8080", output)

	// test: endpoint contains ':' & portnum is NaN
	require.PanicsWithError(
		t,
		"port abc in endpoint config is NaN",
		func() { formatEndpoint("my.valid.host:abc") },
		"Should panic due to port number NaN",
	)

	// test: endpoint contains ':' & portnum is outside RFC range (0-65363)
	require.NotPanics(t, func() { formatEndpoint("my.valid.host:3000") }, "Should NOT panic since port number is valid")
	require.PanicsWithError(
		t,
		"port -12 not in valid RFC range (0-65363)",
		func() { formatEndpoint("my.valid.host:-12") },
		"Should panic due to invalid port number",
	)
	require.PanicsWithError(
		t,
		"port 70123 not in valid RFC range (0-65363)",
		func() { formatEndpoint("my.valid.host:70123") },
		"Should panic due to invalid port number",
	)
}

func TestGetOriginsAsSlice(t *testing.T) {
	t.Run("empty origins env must return empty slice", func(t *testing.T) {
		assert.Empty(t, getOriginsAsSlice(""))
	})

	t.Run("* origins env must return slice with single entry", func(t *testing.T) {
		result := getOriginsAsSlice("*")
		assert.NotEmpty(t, result)
		assert.Len(t, result, 1)
		assert.Equal(t, "*", result[0])
	})

	t.Run("multiple origins env must be separated by commas", func(t *testing.T) {
		result := getOriginsAsSlice("https://example1.com,https://example2.com")
		assert.NotEmpty(t, result)
		assert.Len(t, result, 2)
		assert.Equal(t, "https://example1.com", result[0])
		assert.Equal(t, "https://example2.com", result[1])

		result = getOriginsAsSlice("https://example1.com , https://example2.com")
		assert.NotEmpty(t, result)
		assert.Len(t, result, 2)
		assert.Equal(t, "https://example1.com", result[0])
		assert.Equal(t, "https://example2.com", result[1])

		result = getOriginsAsSlice("https://example1.com https://example2.com")
		assert.NotEmpty(t, result)
		assert.Len(t, result, 1)
		assert.Equal(t, "https://example1.com https://example2.com", result[0])
	})
}

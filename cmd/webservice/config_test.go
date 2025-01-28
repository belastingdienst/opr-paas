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
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
	// add fields if needed
}

func (s *ConfigTestSuite) SetupTest() {
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) TestCorsConfiguration() {

	s.T().Run("must not validate with empty AllowAllOrigins and empty AllowedOrigin", func(t *testing.T) {
		config := NewWSConfig()

		assert.Empty(t, config.AllowAllOrigins)
		assert.Empty(t, config.AllowedOrigin)

		valid, msg := config.Validate()
		assert.False(t, valid)
		assert.Equal(t, "must specify an origin if allowAllOrigins is not set to true", msg)

	})

	s.T().Run("must validate with AllowAllOrigins is true", func(t *testing.T) {
		config := NewWSConfig()

		config.AllowAllOrigins = "true"
		assert.Empty(t, config.AllowedOrigin)

		valid, msg := config.Validate()
		assert.True(t, valid)
		assert.Equal(t, "no issues detected", msg)

		config.AllowedOrigin = "http://www.example.com"
		assert.NotEmpty(t, config.AllowedOrigin)

		valid, msg = config.Validate()
		assert.Equal(t, "no issues detected", msg)

	})

	s.T().Run("must validate with AllowAllOrigins not true and AllowedOrigin set", func(t *testing.T) {
		config := NewWSConfig()
		config.AllowedOrigin = "http://www.example.com"

		assert.Empty(t, config.AllowAllOrigins)
		assert.NotEmpty(t, config.AllowedOrigin)

		valid, msg := config.Validate()
		assert.True(t, valid)
		assert.Equal(t, "no issues detected", msg)
	})
}

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
	require.Panics(t, func() { formatEndpoint("abcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghij:3000") }, "Should panic because hostname is too long")

	// test: endpoint contains ':' & hostname is invalid
	require.NotPanics(t, func() { formatEndpoint("abc.DEF:3000") }, "Should NOT panic because hostname is valid (sanity check)")
	require.NotPanics(t, func() { formatEndpoint("abc.DEF") }, "Should NOT panic because hostname is valid (sanity check)")
	require.NotPanics(t, func() { formatEndpoint("abc.DEF.nl") }, "Should NOT panic because hostname is valid (sanity check)")
	require.NotPanics(t, func() { formatEndpoint("abc") }, "Should NOT panic because hostname is valid (sanity check)")
	require.Panics(t, func() { formatEndpoint("abc#DEF:3000") }, "Should panic because hostname contains illegal character (#)")
	assert.Panics(t, func() { formatEndpoint(".abcDEF:3000") }, "Should panic because hostname starts with illegal character (.)")
	assert.Panics(t, func() { formatEndpoint("ab..cDEF:3000") }, "Should panic because hostname contains double dot character (..)")
	assert.Panics(t, func() { formatEndpoint("-abcDEF:3000") }, "Should panic because hostname starts with illegal character (-)")
	assert.Panics(t, func() { formatEndpoint("abc.DEF-:3000") }, "Should panic because hostname ends with illegal character (-)")
	assert.Panics(t, func() { formatEndpoint("abc.DEF.a:3000") }, "Should panic because hostname TLD too short (<2)")
	assert.Panics(t, func() { formatEndpoint("abc.DEF.666:3000") }, "Should panic because hostname TLD contains illegal character (666)")
	assert.NotPanics(t, func() { formatEndpoint("abc.DEF-ghi.net:3000") }, "Should NOT panic because hostname contains LEGAL character (-)")
	assert.Panics(t, func() { formatEndpoint("abc.DEF_ghi.net:3000") }, "Should panic because hostname contains illegal character (_)")

	// test: endpoint contains ':' & portnum is empty
	output = formatEndpoint("my.valid.host:")
	assert.NotNil(t, output)
	assert.Equal(t, "my.valid.host:8080", output)

	// test: endpoint contains ':' & portnum is NaN
	require.PanicsWithError(t, "port abc in endpoint config is NaN", func() { formatEndpoint("my.valid.host:abc") }, "Should panic due to port number NaN")

	// test: endpoint contains ':' & portnum is outside RFC range (0-65363)
	require.NotPanics(t, func() { formatEndpoint("my.valid.host:3000") }, "Should NOT panic since port number is valid")
	require.PanicsWithError(t, "port -12 not in valid RFC range (0-65363)", func() { formatEndpoint("my.valid.host:-12") }, "Should panic due to invalid port number")
	require.PanicsWithError(t, "port 70123 not in valid RFC range (0-65363)", func() { formatEndpoint("my.valid.host:70123") }, "Should panic due to invalid port number")
}

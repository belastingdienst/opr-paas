/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	v "github.com/belastingdienst/opr-paas/internal/version"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function for testing
func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func Test_getConfig(t *testing.T) {
	config := getConfig()

	assert.NotNil(t, config)
	assert.Len(t, config.AdminApiKey, 64)

	assert.NotNil(t, config.Endpoint)
	assert.Equal(t, ":8080", config.Endpoint)

	assert.NotNil(t, config.PublicKeyPath)
	assert.Equal(t, "/secrets/paas/publicKey", config.PublicKeyPath)

	_config = &WSConfig{
		PublicKeyPath: "/some/weird/path",
		Endpoint:      ":3000",
		AdminApiKey:   "dkBKevKlUEveMirzLzFdDVhplnPzSKPtrPphWOPXjuGKfFjCTHaySNGGnaIBPiEJ",
	}

	config = getConfig()

	assert.NotNil(t, config)
	assert.Len(t, config.AdminApiKey, 64)
	assert.Equal(t, "dkBKevKlUEveMirzLzFdDVhplnPzSKPtrPphWOPXjuGKfFjCTHaySNGGnaIBPiEJ", config.AdminApiKey)

	assert.NotNil(t, config.Endpoint)
	assert.Equal(t, ":3000", config.Endpoint)

	assert.NotNil(t, config.PublicKeyPath)
	assert.Equal(t, "/some/weird/path", config.PublicKeyPath)
}

func Test_getRSA(t *testing.T) {
	// test: non-existing public key should panic
	getConfig()
	_config.PublicKeyPath = "/random/non-existing/public/keyfile"
	assert.Equal(t, "/random/non-existing/public/keyfile", _config.PublicKeyPath)
	assert.Nil(t, _crypt)
	assert.Panics(t, func() { getRsa("paasName") }, "Failed to panic using non-existing public key")

	// reset
	_crypt = nil
	_config = nil

	// test: non-existing _crypt results in single entry _crypt
	getConfig()
	_config.PublicKeyPath = "../../testdata/public.rsa.key"
	assert.Nil(t, _crypt)
	output := getRsa("paasName")
	assert.Len(t, _crypt, 1)
	assert.IsType(t, &crypt.Crypt{}, output)
	assert.IsType(t, map[string]*crypt.Crypt{}, _crypt)

	encrypted, err := output.Encrypt([]byte("My test string"))
	require.NoError(t, err)
	assert.Len(t, encrypted, 684)

	// explicitely didn't reset

	// test: results in two entries in _crypt
	getConfig()
	_config.PublicKeyPath = "../../testdata/public.rsa.key"
	assert.NotNil(t, _crypt)
	output = getRsa("paasName2")
	assert.Len(t, _crypt, 2)
	assert.IsType(t, &crypt.Crypt{}, output)
	assert.IsType(t, map[string]*crypt.Crypt{}, _crypt)

	encrypted, err = output.Encrypt([]byte("My second string"))
	require.NoError(t, err)
	assert.Len(t, encrypted, 684)
}

func Test_version(t *testing.T) {
	expected := gin.H{
		"version": v.PAAS_VERSION,
	}

	router := SetupRouter()
	w := performRequest(router, "GET", "/version")
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal([]byte(w.Body.Bytes()), &response)
	value, exists := response["version"]

	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, expected["version"], value)
}

func Test_healthz(t *testing.T) {
	expected := gin.H{
		"message": "healthy",
	}

	router := SetupRouter()
	w := performRequest(router, "GET", "/healthz")
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal([]byte(w.Body.Bytes()), &response)
	value, exists := response["message"]

	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, expected["message"], value)
}

func Test_readyz(t *testing.T) {
	expected := gin.H{
		"message": "ready",
	}

	router := SetupRouter()
	w := performRequest(router, "GET", "/readyz")
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal([]byte(w.Body.Bytes()), &response)
	value, exists := response["message"]

	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, expected["message"], value)
}

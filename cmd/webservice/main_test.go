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
	"os"
	"strings"
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/crypt"
	v "github.com/belastingdienst/opr-paas/internal/version"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Helper function for testing
func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func Test_getConfig(t *testing.T) {
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, "*")

	// Reset config if any test before set config
	_config = nil
	_crypt = nil
	config := getConfig()

	assert.NotNil(t, config)
	assert.NotNil(t, config.Endpoint)
	assert.Equal(t, ":8080", config.Endpoint)

	assert.NotNil(t, config.PublicKeyPath)
	assert.Equal(t, "/secrets/paas/publicKey", config.PublicKeyPath)

	_config = &WSConfig{
		PublicKeyPath:  "/some/weird/path",
		Endpoint:       ":3000",
		AllowedOrigins: []string{"http://example.com"},
	}

	config = getConfig()

	assert.NotNil(t, config)
	assert.NotNil(t, config.Endpoint)
	assert.Equal(t, ":3000", config.Endpoint)

	assert.NotNil(t, config.PublicKeyPath)
	assert.Equal(t, "/some/weird/path", config.PublicKeyPath)
}

func Test_getRSA(t *testing.T) {
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, "*")

	// Reset config if any test before set config
	_config = nil
	_crypt = nil
	getConfig()

	// generate private/public keys
	t.Log("creating temp private key")
	priv, err := os.CreateTemp("", "private")
	require.NoError(t, err, "Creating tempfile for private key")
	defer os.Remove(priv.Name()) // clean up

	t.Log("creating temp public key")
	pub, err := os.CreateTemp("", "public")
	require.NoError(t, err, "Creating tempfile for public key")
	defer os.Remove(pub.Name()) // clean up

	t.Log("generating new keys and creating crypt")
	crypt.GenerateKeyPair(priv.Name(), pub.Name()) //nolint:errcheck // this is fine in test

	// test: non-existing public key should panic
	getConfig()
	_config.PublicKeyPath = "/random/non-existing/public/keyfile"
	assert.Equal(t, "/random/non-existing/public/keyfile", _config.PublicKeyPath)
	assert.Nil(t, _crypt)
	t.Log("getting paasName 1")
	assert.Panics(t, func() { getRsa("paasName") }, "Failed to panic using non-existing public key")

	// reset
	_crypt = nil
	_config = nil

	// test: non-existing _crypt results in single entry _crypt
	getConfig()
	_config.PublicKeyPath = pub.Name()
	_config.PrivateKeyPath = priv.Name()
	assert.Nil(t, _crypt)
	t.Log("getting paasName 2")
	output := getRsa("paasName")
	assert.Len(t, _crypt, 1)
	assert.IsType(t, &crypt.Crypt{}, output)
	assert.IsType(t, map[string]*crypt.Crypt{}, _crypt)

	encrypted, err := output.Encrypt([]byte("My test string"))
	require.NoError(t, err)
	assert.Len(t, encrypted, 684)

	// explicitly didn't reset

	// test: results in two entries in _crypt
	getConfig()
	_config.PublicKeyPath = pub.Name()
	assert.NotNil(t, _crypt)
	t.Log("getting paasName2")
	output = getRsa("paasName2")
	assert.Len(t, _crypt, 2)
	assert.IsType(t, &crypt.Crypt{}, output)
	assert.IsType(t, map[string]*crypt.Crypt{}, _crypt)

	encrypted, err = output.Encrypt([]byte("My second string"))
	require.NoError(t, err)
	assert.Len(t, encrypted, 684)
}

func TestNoSniffIsSet(t *testing.T) {
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, "*")

	router := SetupRouter()
	w := performRequest(router, "GET", "/version")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

func Test_version(t *testing.T) {
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, "*")

	expected := gin.H{
		"version": v.PaasVersion,
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
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, "*")

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
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, "*")

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

func Test_v1CheckPaas(t *testing.T) {
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, "*")

	// Reset config if any test before set config
	_config = nil
	_crypt = nil

	priv, err := os.CreateTemp("", "private")
	require.NoError(t, err, "Creating tempfile for private key")
	defer os.Remove(priv.Name()) // clean up

	pub, err := os.CreateTemp("", "public")
	require.NoError(t, err, "Creating tempfile for public key")
	defer os.Remove(pub.Name()) // clean up

	// Set env for ws Config
	t.Setenv("PAAS_PUBLIC_KEY_PATH", pub.Name())    //nolint:errcheck // this is fine in test
	t.Setenv("PAAS_PRIVATE_KEYS_PATH", priv.Name()) //nolint:errcheck // this is fine in test

	// Generate keyPair to be used during test
	crypt.GenerateKeyPair(priv.Name(), pub.Name()) //nolint:errcheck // this is fine in test

	// Encrypt secret for test
	rsa := getRsa("testPaas")

	encrypted, err := rsa.Encrypt([]byte("My test string"))
	require.NoError(t, err)

	getConfig()
	router := SetupRouter()

	w := httptest.NewRecorder()

	validRequest := RestCheckPaasInput{
		Paas: v1alpha1.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testPaas",
			},
			Spec: v1alpha1.PaasSpec{
				SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": encrypted},
				Capabilities: v1alpha1.PaasCapabilities{
					"sso": v1alpha1.PaasCapability{Enabled: true, SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": encrypted}},
				},
			},
		},
	}
	checkPaasJson, _ := json.Marshal(validRequest)

	req, _ := http.NewRequest("POST", "/v1/checkpaas", strings.NewReader(string(checkPaasJson)))
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	response := RestCheckPaasResult{
		PaasName:  "testPaas",
		Decrypted: true,
		Error:     "",
	}
	responseJson, _ := json.MarshalIndent(response, "", "    ")
	assert.JSONEq(t, string(responseJson), w.Body.String())

	// Reset recorder
	w = httptest.NewRecorder()

	invalidRequest := RestCheckPaasInput{
		Paas: v1alpha1.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testPaas2",
			},
			Spec: v1alpha1.PaasSpec{
				SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": "ZW5jcnlwdGVkCg=="},
				Capabilities: v1alpha1.PaasCapabilities{
					"sso": v1alpha1.PaasCapability{Enabled: true, SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": encrypted}},
				},
			},
		},
	}
	invalidCheckPaasJson, _ := json.Marshal(invalidRequest)

	req2, _ := http.NewRequest("POST", "/v1/checkpaas", strings.NewReader(string(invalidCheckPaasJson)))
	router.ServeHTTP(w, req2)

	assert.Equal(t, 422, w.Code)
	response2 := RestCheckPaasResult{
		PaasName:  "testPaas2",
		Decrypted: false,
		Error:     "testPaas2: .spec.sshSecrets[ssh://git@scm/some-repo.git], error: unable to decrypt data with any of the private keys , testPaas2: .spec.capabilities[sso].sshSecrets[ssh://git@scm/some-repo.git], error: unable to decrypt data with any of the private keys",
	}
	response2Json, _ := json.MarshalIndent(response2, "", "    ")
	assert.JSONEq(t, string(response2Json), w.Body.String())
}

func Test_v1CheckPaasInternalServerError(t *testing.T) {
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, "*")

	// Reset config if any test before get config
	_config = nil
	_crypt = nil
	getConfig()

	router := SetupRouter()

	w := httptest.NewRecorder()

	validRequest := RestCheckPaasInput{
		Paas: v1alpha1.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testPaas",
			},
			Spec: v1alpha1.PaasSpec{
				SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": "ZW5jcnlwdGVkCg=="},
			},
		},
	}
	checkPaasJson, _ := json.Marshal(validRequest)

	req, _ := http.NewRequest("POST", "/v1/checkpaas", strings.NewReader(string(checkPaasJson)))
	router.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)
	assert.Equal(t, "", w.Body.String())
}

func TestBuildCSP(t *testing.T) {
	externalHosts := ""
	expected := "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; font-src 'self'; object-src 'none'"
	assert.Equal(t, expected, buildCSP(externalHosts))

	externalHosts = "http://example.com"
	expected = "default-src 'none'; script-src 'self' http://example.com; style-src 'self' http://example.com; img-src 'self' http://example.com; connect-src 'self' http://example.com; font-src 'self' http://example.com; object-src 'none'"
	assert.Equal(t, expected, buildCSP(externalHosts))
}

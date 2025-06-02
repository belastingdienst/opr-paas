/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	v "github.com/belastingdienst/opr-paas/internal/version"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	allowedOriginsVal = "*"
	rsaKeySize        = 2048
	testPaasName      = "testPaas"
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
	t.Setenv(allowedOriginsEnv, allowedOriginsVal)

	// Reset config if any test before set config
	_config = nil
	_crypt = nil
	config := getConfig()

	assert.NotNil(t, config)
	assert.NotNil(t, config.Endpoint)
	assert.Equal(t, ":8080", config.Endpoint)

	assert.NotNil(t, config.PublicKeyPath)
	assert.Equal(t, "/secrets/paas/publicKey", config.PublicKeyPath)

	_config = &wsConfig{
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
	t.Setenv(allowedOriginsEnv, allowedOriginsVal)

	// Reset config if any test before set config
	_config = nil
	_crypt = nil
	getConfig()

	pub, priv, toDefer := makeCrypt(t)
	defer toDefer()

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
	t.Setenv(allowedOriginsEnv, allowedOriginsVal)

	router := setupRouter()
	w := performRequest(router, "GET", "/version")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

func Test_version(t *testing.T) {
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, allowedOriginsVal)

	expected := gin.H{
		"version": v.PaasVersion,
	}

	router := setupRouter()
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
	t.Setenv(allowedOriginsEnv, allowedOriginsVal)

	expected := gin.H{
		"message": "healthy",
	}

	router := setupRouter()
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
	t.Setenv(allowedOriginsEnv, allowedOriginsVal)

	expected := gin.H{
		"message": "ready",
	}

	router := setupRouter()
	w := performRequest(router, "GET", "/readyz")
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal([]byte(w.Body.Bytes()), &response)
	value, exists := response["message"]

	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, expected["message"], value)
}

//revive:disable-next-line
func Test_v1CheckPaas(t *testing.T) {
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, allowedOriginsVal)

	// Reset config if any test before set config
	_config = nil
	_crypt = nil

	_, _, toDefer := makeCrypt(t)
	defer toDefer()

	// Encrypt secret for test
	retrievedRsa := getRsa(testPaasName)

	encrypted, err := retrievedRsa.Encrypt([]byte("My test string"))
	require.NoError(t, err)

	getConfig()
	router := setupRouter()

	w := httptest.NewRecorder()

	validRequest := RestCheckPaasInput{
		Paas: v1alpha1.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: testPaasName,
			},
			Spec: v1alpha1.PaasSpec{
				SSHSecrets: map[string]string{"ssh://git@scm/some-repo.git": encrypted},
				Capabilities: v1alpha1.PaasCapabilities{
					"sso": v1alpha1.PaasCapability{
						Enabled:    true,
						SSHSecrets: map[string]string{"ssh://git@scm/some-repo.git": encrypted},
					},
				},
			},
		},
	}
	checkPaasJSON, _ := json.Marshal(validRequest)

	req, _ := http.NewRequest("POST", "/v1/checkpaas", strings.NewReader(string(checkPaasJSON)))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := RestCheckPaasResult{
		PaasName:  testPaasName,
		Decrypted: true,
		Error:     "",
	}
	responseJSON, _ := json.MarshalIndent(response, "", "    ")
	assert.JSONEq(t, string(responseJSON), w.Body.String())

	// Reset recorder
	w = httptest.NewRecorder()

	invalidRequest := RestCheckPaasInput{
		Paas: v1alpha1.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testPaas2",
			},
			Spec: v1alpha1.PaasSpec{
				SSHSecrets: map[string]string{"ssh://git@scm/some-repo.git": "ZW5jcnlwdGVkCg=="},
				Capabilities: v1alpha1.PaasCapabilities{
					"sso": v1alpha1.PaasCapability{
						Enabled:    true,
						SSHSecrets: map[string]string{"ssh://git@scm/some-repo.git": encrypted},
					},
				},
			},
		},
	}
	invalidcheckPaasJSON, _ := json.Marshal(invalidRequest)

	req2, _ := http.NewRequest("POST", "/v1/checkpaas", strings.NewReader(string(invalidcheckPaasJSON)))
	router.ServeHTTP(w, req2)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	response2 := RestCheckPaasResult{
		PaasName:  "testPaas2",
		Decrypted: false,
		// revive:disable-next-line
		Error: "testPaas2: .spec.sshSecrets[ssh://git@scm/some-repo.git], error: unable to decrypt data with any of the private keys , testPaas2: .spec.capabilities[sso].sshSecrets[ssh://git@scm/some-repo.git], error: unable to decrypt data with any of the private keys",
	}
	response2JSON, _ := json.MarshalIndent(response2, "", "    ")
	assert.JSONEq(t, string(response2JSON), w.Body.String())
}

func generateRSAPrivateKeyPEM(bits int) (string, error) {
	// Generate the RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", err
	}

	// Encode the private key to PKCS#1 ASN.1 PEM
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	})

	return string(privPEM), nil
}

//revive:disable-next-line
func Test_v1Encrypt(t *testing.T) {
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, allowedOriginsVal)

	// Reset config if any test before set config
	_config = nil
	_crypt = nil

	_, _, toDefer := makeCrypt(t)
	defer toDefer()

	getConfig()
	router := setupRouter()

	w := httptest.NewRecorder()

	privKey, err := generateRSAPrivateKeyPEM(rsaKeySize)
	if err != nil {
		panic(err)
	}

	validRequest := RestEncryptInput{
		PaasName: testPaasName,
		Secret:   privKey,
	}
	encryptJson, _ := json.Marshal(validRequest)

	req, _ := http.NewRequest("POST", "/v1/encrypt", strings.NewReader(string(encryptJson)))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseObject RestEncryptResult
	err = json.Unmarshal(w.Body.Bytes(), &responseObject)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, testPaasName, responseObject.PaasName)
	assert.True(t, responseObject.Valid)
	assert.NotEmpty(t, responseObject.Encrypted)

	// Reset recorder
	w = httptest.NewRecorder()

	invalidRequest := RestEncryptInput{
		PaasName: "testPaas2",
		Secret:   "invalidPrivateKey",
	}

	invalidEncryptjson, _ := json.Marshal(invalidRequest)

	req2, _ := http.NewRequest("POST", "/v1/encrypt", strings.NewReader(string(invalidEncryptjson)))
	router.ServeHTTP(w, req2)

	assert.Equal(t, http.StatusOK, w.Code)
	response2 := RestEncryptResult{
		PaasName:  "testPaas2",
		Valid:     false,
		Encrypted: "",
	}
	response2JSON, _ := json.MarshalIndent(response2, "", "    ")
	assert.JSONEq(t, string(response2JSON), w.Body.String())
}

func Test_v1CheckPaasInternalServerError(t *testing.T) {
	// Allow all origins for test
	t.Setenv(allowedOriginsEnv, allowedOriginsVal)

	// Reset config if any test before get config
	_config = nil
	_crypt = nil
	getConfig()

	router := setupRouter()

	w := httptest.NewRecorder()

	validRequest := RestCheckPaasInput{
		Paas: v1alpha1.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: testPaasName,
			},
			Spec: v1alpha1.PaasSpec{
				SSHSecrets: map[string]string{"ssh://git@scm/some-repo.git": "ZW5jcnlwdGVkCg=="},
			},
		},
	}
	checkPaasJSON, _ := json.Marshal(validRequest)

	req, _ := http.NewRequest("POST", "/v1/checkpaas", strings.NewReader(string(checkPaasJSON)))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "", w.Body.String())
}

func TestBuildCSP(t *testing.T) {
	externalHosts := ""
	// revive:disable-next-line
	expected := "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; font-src 'self'; object-src 'none'"
	assert.Equal(t, expected, buildCSP(externalHosts))

	externalHosts = "http://example.com"
	// revive:disable-next-line
	expected = "default-src 'none'; script-src 'self' http://example.com; style-src 'self' http://example.com; img-src 'self' http://example.com; connect-src 'self' http://example.com; font-src 'self' http://example.com; object-src 'none'"
	assert.Equal(t, expected, buildCSP(externalHosts))
}

func makeCrypt(t *testing.T) (pub, priv *os.File, toDefer func()) {
	// generate private/public keys
	t.Log("creating temp private key")
	priv, err := os.CreateTemp("", "private")
	require.NoError(t, err, "Creating tempfile for private key")

	t.Log("creating temp public key")
	pub, err = os.CreateTemp("", "public")
	require.NoError(t, err, "Creating tempfile for public key")

	// Set env for ws Config
	t.Setenv("PAAS_PUBLIC_KEY_PATH", pub.Name())    //nolint:errcheck // this is fine in test
	t.Setenv("PAAS_PRIVATE_KEYS_PATH", priv.Name()) //nolint:errcheck // this is fine in test

	// Generate keyPair to be used during test
	crypt.GenerateKeyPair(priv.Name(), pub.Name()) //nolint:errcheck // this is fine in test

	return pub, priv, func() { // clean up function
		os.Remove(priv.Name())
		os.Remove(pub.Name())
	}
}

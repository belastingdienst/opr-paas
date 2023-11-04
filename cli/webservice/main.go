package main

/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"fmt"
	"net/http"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/belastingdienst/opr-paas/internal/version"
	"github.com/gin-gonic/gin"
)

var (
	_crypt  map[string]*crypt.Crypt
	_config *WSConfig
)

func getConfig() *WSConfig {
	if _config == nil {
		config := NewWSConfig()
		_config = &config
	}
	return _config
}

func getRsa(paas string) *crypt.Crypt {
	if _crypt == nil {
		_crypt = make(map[string]*crypt.Crypt)
	}
	config := getConfig()
	if c, exists := _crypt[paas]; !exists {
		c = crypt.NewCrypt(config.PrivateKeyPath, config.PublicKeyPath, paas)
		_crypt[paas] = c
		return c
	} else {
		return c
	}
}

// getVersion returns the operator version this webservice is built for
func getVersion(c *gin.Context) {
	output := RestVersionResult{
		Version: version.PAAS_VERSION,
	}
	c.IndentedJSON(http.StatusOK, output)
}

// getEncrypt encrypts a secret and returns the encrypted value
func getEncrypt(c *gin.Context) {
	var input RestInput
	if err := c.BindJSON(&input); err != nil {
		return
	}
	secret := []byte(input.Secret)
	if encrypted, err := getRsa(input.PaasName).Encrypt(secret); err != nil {
		return
	} else {
		output := RestEncryptResult{
			PaasName:  input.PaasName,
			Encrypted: encrypted,
		}
		c.IndentedJSON(http.StatusOK, output)
	}
}

// getAlbums responds with the list of all albums as JSON.
func getGenerate(c *gin.Context) {
	var output RestGenerateResult
	config := getConfig()
	if private, public, err := crypt.NewCrypt(config.PrivateKeyPath, config.PublicKeyPath, "").GenerateStrings(); err != nil {
		c.AbortWithError(http.StatusFailedDependency, fmt.Errorf("could not create a new crypt to generate new keys: %e", err))
		return
	} else {
		output.Private = private
		output.Public = public
	}
	c.IndentedJSON(http.StatusOK, output)
}

func main() {
	router := gin.Default()
	router.GET("/version", getVersion)
	router.GET("/encrypt", getEncrypt)
	router.GET("/generate", getGenerate)
	router.Run()
}

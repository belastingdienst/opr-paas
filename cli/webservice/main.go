/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	_version "github.com/belastingdienst/opr-paas/internal/version"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
		c = crypt.NewCrypt("", config.PublicKeyPath, paas)
		_crypt[paas] = c
		return c
	} else {
		return c
	}
}

// getEncrypt encrypts a secret and returns the encrypted value
func v1Encrypt(c *gin.Context) {
	var input RestEncryptInput
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

// v1Generate generates a new keypair to be used by the the PaaS operator
// Only to be used by PaaS administrators
func v1Generate(c *gin.Context) {
	var input RestGenerateInput
	if err := c.BindJSON(&input); err != nil {
		return
	}
	if input.ApiKey != getConfig().AdminApiKey {
		return
	}
	var output RestGenerateResult
	if private, public, err := crypt.NewCrypt("", "", "").GenerateStrings(); err != nil {
		c.AbortWithError(http.StatusFailedDependency, fmt.Errorf("could not create a new crypt to generate new keys: %e", err))
		return
	} else {
		output.Private = private
		output.Public = public
	}
	c.IndentedJSON(http.StatusOK, output)
}

// version returns the operator version this webservice is built for
func version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": _version.PAAS_VERSION,
	})
}

// healthz is a liveness probe.
func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "healthy",
	})
}

// readyz is a readiness probe.
func readyz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "ready",
	})
}

func main() {
	log.Println("Starting API endpoint")
	log.Printf("Version: %s", _version.PAAS_VERSION)
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(
		gin.LoggerWithWriter(gin.DefaultWriter, "/healthz", "/readyz"),
		gin.Recovery(),
	)

	router.SetTrustedProxies(nil)
	router.GET("/version", version)
	router.POST("/v1/encrypt", v1Encrypt)
	router.GET("/v1/generate", v1Generate)
	router.GET("/healthz", healthz)
	router.GET("/readyz", readyz)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	ep := getConfig().Endpoint
	log.Printf("Listening on: %s", ep)
	router.Run(ep)
}

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
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/ssh"
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
	if c, exists := _crypt[paas]; exists {
		return c
	} else if c, err := crypt.NewCrypt([]string{}, config.PublicKeyPath, paas); err != nil {
		panic(fmt.Errorf("unable to create a crypt: %e", err))
	} else {
		_crypt[paas] = c
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
	if _, err := ssh.ParsePrivateKey(secret); err == nil {
		if encrypted, err := getRsa(input.PaasName).Encrypt(secret); err != nil {
			return
		} else {
			output := RestEncryptResult{
				PaasName:  input.PaasName,
				Encrypted: encrypted,
				Valid:     true,
			}
			c.IndentedJSON(http.StatusOK, output)
		}
	} else {
		output := RestEncryptResult{
			PaasName:  input.PaasName,
			Encrypted: "",
			Valid:     false,
		}
		c.IndentedJSON(http.StatusOK, output)
	}
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
	router := gin.Default()
	// - No origin allowed by default
	// - GET,POST, PUT, HEAD methods
	// - Credentials share disabled
	// - Preflight requests cached for 12 hours
	config := cors.DefaultConfig()
	config.AllowMethods = []string{"GET", "POST"}
	// config.AllowOrigins = []string{"http://bla.com"}
	config.AllowAllOrigins = true

	router.Use(
		cors.New(config),
		gin.LoggerWithWriter(gin.DefaultWriter, "/healthz", "/readyz"),
		gin.Recovery(),
	)

	router.SetTrustedProxies(nil)
	router.GET("/version", version)
	router.POST("/v1/encrypt", v1Encrypt)
	router.GET("/healthz", healthz)
	router.GET("/readyz", readyz)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	ep := getConfig().Endpoint
	log.Printf("Listening on: %s", ep)
	router.Run(ep)
}

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
	"strings"
	"sync"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/belastingdienst/opr-paas/internal/utils"
	_version "github.com/belastingdienst/opr-paas/internal/version"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/ssh"
)

var (
	_crypt     map[string]*crypt.Crypt
	_cryptLock sync.RWMutex
	_config    *WSConfig
	_fw        *utils.FileWatcher
)

func getConfig() *WSConfig {
	if _config == nil {
		config := NewWSConfig()
		_config = &config
	}

	return _config
}

func resetRsa() {
	log.Println("Resetting RSA")
	_cryptLock.Lock()
	defer _cryptLock.Unlock()
	_crypt = make(map[string]*crypt.Crypt)
}

func getCrypt(paas string) *crypt.Crypt {
	_cryptLock.RLock()
	defer _cryptLock.RUnlock()
	if c, exists := _crypt[paas]; exists {
		return c
	}
	return nil
}

func getRsa(paas string) *crypt.Crypt {
	config := getConfig()
	if _fw == nil {
		log.Println("Starting watcher")
		_fw = utils.NewFileWatcher(config.PrivateKeyPath, config.PublicKeyPath)
	}
	// It is crucial that we have this first and nil check on _crypt later
	if _fw.WasTriggered() {
		log.Println("Files changed")
		resetRsa()
	} else if _crypt == nil {
		log.Println("crypt empty")
		resetRsa()
	}
	if c := getCrypt(paas); c != nil {
		return c
	}

	c, err := crypt.NewCryptFromFiles([]string{config.PrivateKeyPath}, config.PublicKeyPath, paas)
	if err != nil {
		panic(fmt.Errorf("unable to create a crypt: %w", err))
	}

	_cryptLock.Lock()
	defer _cryptLock.Unlock()
	_crypt[paas] = c

	return c
}

// v1Encrypt encrypts a secret and returns the encrypted value
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

// v1CheckPaas checks whether a Paas can be decrypted using provided private/public keys
func v1CheckPaas(c *gin.Context) {
	var input RestCheckPaasInput
	if err := c.BindJSON(&input); err != nil {
		c.IndentedJSON(http.StatusBadRequest, RestCheckPaasResult{"", false, err.Error()})
		return
	}
	rsa := getRsa(input.Paas.Name)
	err := CheckPaas(rsa, &input.Paas)
	if err != nil {
		if strings.Contains(err.Error(), "unable to decrypt data with any of the private keys") ||
			strings.Contains(err.Error(), "base64") {
			output := RestCheckPaasResult{
				PaasName:  input.Paas.Name,
				Decrypted: false,
				Error:     err.Error(),
			}
			c.IndentedJSON(http.StatusUnprocessableEntity, output)
			return
		} else {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	} else {
		output := RestCheckPaasResult{
			PaasName:  input.Paas.Name,
			Decrypted: true,
			Error:     "",
		}
		c.IndentedJSON(http.StatusOK, output)
	}
}

// version returns the operator version this webservice is built for
func version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": _version.PaasVersion,
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

func SetupRouter() *gin.Engine {
	router := gin.New()

	// CORS
	// Use default config as base
	config := cors.DefaultConfig()

	// Override default config where needed
	config.AllowMethods = []string{"GET", "POST", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type"}
	// Ensure closed default
	config.AllowAllOrigins = false
	config.AllowOrigins = nil
	if len(getConfig().AllowedOrigins) > 0 {
		if len(getConfig().AllowedOrigins) == 1 && getConfig().AllowedOrigins[0] == "*" {
			config.AllowAllOrigins = true
		} else {
			config.AllowOrigins = getConfig().AllowedOrigins
		}
	}

	if err := config.Validate(); err != nil {
		panic(fmt.Errorf("cors config invalid: %w", err))
	}

	router.Use(
		cors.New(config),
		gin.LoggerWithWriter(gin.DefaultWriter, "/healthz", "/readyz"),
		gin.Recovery(),
	)

	err := router.SetTrustedProxies(nil)
	if err != nil {
		panic(fmt.Errorf("setTrustedProxies %w", err))
	}

	// Insert the X-Content-Type-Options and CSP headers.
	router.Use(func(c *gin.Context) {
		cspString := buildCSP(strings.Join(getConfig().AllowedOrigins, " "))
		c.Header("Content-Security-Policy", cspString)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	})

	router.GET("/version", version)
	router.POST("/v1/encrypt", v1Encrypt)
	router.POST("/v1/checkpaas", v1CheckPaas)
	router.GET("/healthz", healthz)
	router.GET("/readyz", readyz)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return router
}

func main() {
	log.Println("Starting API endpoint")
	log.Printf("Version: %s", _version.PaasVersion)
	gin.SetMode(gin.ReleaseMode)

	router := SetupRouter()

	ep := getConfig().Endpoint
	log.Printf("Listening on: %s", ep)
	err := router.Run(ep)
	if err != nil {
		panic(fmt.Errorf("router go boom: %w", err))
	}
}

// buildCSP returns a Content-Security-Policy string.
// If externalHosts is non-empty, we append it to script-src, style-src, etc.
// externalHosts should a space-separated list of http:// and/or https:// urls
func buildCSP(externalHosts string) string {
	defaultSrc := "default-src 'none'"
	scriptSrc := "script-src 'self'"
	styleSrc := "style-src 'self'"
	imgSrc := "img-src 'self'"
	connectSrc := "connect-src 'self'"
	fontSrc := "font-src 'self'"
	objectSrc := "object-src 'none'"

	// If we have a non-empty external host, append it to each directive that needs it.
	if externalHosts != "" {
		scriptSrc += " " + externalHosts
		styleSrc += " " + externalHosts
		imgSrc += " " + externalHosts
		connectSrc += " " + externalHosts
		fontSrc += " " + externalHosts
	}

	// Combine them into one directive string
	return fmt.Sprintf(
		"%s; %s; %s; %s; %s; %s; %s",
		defaultSrc, scriptSrc, styleSrc, imgSrc, connectSrc, fontSrc, objectSrc,
	)
}

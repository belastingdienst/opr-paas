/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	publicEnv         = "PAAS_PUBLIC_KEY_PATH"
	defaultPublicPath = "/secrets/paas/publicKey"
	endpointEnv       = "PAAS_ENDPOINT"
	defaultEndpoint   = ":8080"
	adminApiKey       = "PAAS_ADMIN_API_KEY"
)

type WSConfig struct {
	PublicKeyPath string
	Endpoint      string
	AdminApiKey   string
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func NewWSConfig() WSConfig {
	var config WSConfig
	config.PublicKeyPath = os.Getenv(publicEnv)
	if config.PublicKeyPath == "" {
		config.PublicKeyPath = defaultPublicPath
	}
	config.Endpoint = os.Getenv(endpointEnv)
	if config.Endpoint == "" {
		config.Endpoint = defaultEndpoint
	} else if strings.Contains(config.Endpoint, ":") {
		parts := strings.Split(config.Endpoint, ":")
		host := parts[0]
		if len(host) > 63 {
			panic(fmt.Errorf("invalid hostname %s longer than 63 characters", host))
		} else {
			if match, err := regexp.MatchString(`[^0-9.a-zA-Z-:]`, host); err != nil {
				panic("invalid regular expression for hostname")
			} else if match {
				panic(fmt.Errorf("invalid hostname %s in endpoint config", host))
			}
		}
		port := parts[1]
		if port == "" {
			host = "8080"
		} else {
			if portNum, err := strconv.Atoi(port); err != nil {
				panic(fmt.Errorf("invalid hostname %s in endpoint config", port))
			} else if portNum < 0 || portNum > 65353 {
				panic(fmt.Errorf("invalid port %s not in valid RFC range (0-65363)", port))
			}
		}
		config.Endpoint = fmt.Sprintf("%s:%s", host, port)
	} else {
		config.Endpoint = fmt.Sprintf("%s:8080", config.Endpoint)
	}
	config.AdminApiKey = os.Getenv(adminApiKey)
	if config.AdminApiKey == "" {
		config.AdminApiKey = randStringBytes(64)
		log.Printf("Generated random Admin API key: %s", config.AdminApiKey)
	}
	return config

}

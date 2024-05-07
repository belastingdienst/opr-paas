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
	publicEnv           = "PAAS_PUBLIC_KEY_PATH"
	defaultPublicPath   = "/secrets/paas/publicKey"
	endpointEnv         = "PAAS_ENDPOINT"
	defaultEndpointPort = 8080
	adminApiKey         = "PAAS_ADMIN_API_KEY"
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

func formatEndpoint(endpoint string) string {
	if endpoint == "" {
		return fmt.Sprintf(":%d", defaultEndpointPort)
	}
	if strings.Contains(endpoint, ":") {
		parts := strings.Split(endpoint, ":")
		host := parts[0]
		if len(host) > 63 {
			panic(fmt.Errorf("invalid hostname %s longer than 63 characters", host))
		}

		// TODO: should this be tighter? For example: (?=^.{4,253}$)(^((?!-)[a-zA-Z0-9-]{1,63}(?<!-)\.)+[a-zA-Z]{2,63}$)
		if match, err := regexp.MatchString(`[^0-9.a-zA-Z-:]`, host); err != nil {
			panic("invalid regular expression for hostname")
		} else if match {
			panic(fmt.Errorf("invalid hostname %s in endpoint config", host))
		}

		port := parts[1]
		if port == "" {
			port = fmt.Sprintf("%d", defaultEndpointPort)
		} else if portNum, err := strconv.Atoi(port); err != nil {
			panic(fmt.Errorf("port %s in endpoint config is NaN", port))
		} else if portNum < 0 || portNum > 65353 {
			panic(fmt.Errorf("port %s not in valid RFC range (0-65363)", port))
		}
		return fmt.Sprintf("%s:%s", host, port)
	}
	return fmt.Sprintf("%s:%d", endpoint, defaultEndpointPort)
}

func NewWSConfig() WSConfig {
	var config WSConfig
	config.PublicKeyPath = os.Getenv(publicEnv)
	if config.PublicKeyPath == "" {
		config.PublicKeyPath = defaultPublicPath
	}
	config.Endpoint = formatEndpoint(os.Getenv(endpointEnv))
	config.AdminApiKey = os.Getenv(adminApiKey)
	if config.AdminApiKey == "" {
		config.AdminApiKey = randStringBytes(64)
		log.Printf("Generated random Admin API key: %s", config.AdminApiKey)
	}
	return config

}

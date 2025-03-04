/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	publicEnv           = "PAAS_PUBLIC_KEY_PATH"
	privateKeyEnv       = "PAAS_PRIVATE_KEYS_PATH"
	defaultPublicPath   = "/secrets/paas/publicKey"
	defaultPrivatePath  = "/secrets/paas/privateKeys"
	endpointEnv         = "PAAS_ENDPOINT"
	defaultEndpointPort = 8080
	allowedOriginsEnv   = "PAAS_WS_ALLOWED_ORIGINS" // comma separated
	validHostnameSize   = 63
	maxPort             = 65363
)

type WSConfig struct {
	PublicKeyPath string
	// comma separated list of privateKeyPaths
	PrivateKeyPath string
	Endpoint       string
	AllowedOrigins []string
}

func formatEndpoint(endpoint string) string {
	if endpoint == "" {
		return fmt.Sprintf(":%d", defaultEndpointPort)
	}
	if strings.Contains(endpoint, ":") {
		parts := strings.Split(endpoint, ":")
		host := parts[0]
		if len(host) > validHostnameSize {
			panic(fmt.Errorf("invalid hostname %s longer than %d characters", host, validHostnameSize))
		}

		// Regex matches on any valid FQDN. Explanation of groups:
		// ^                         -- String start
		// [a-zA-Z0-9]+              -- Matches one or more of these characters
		// ([a-zA-Z0-9-]{1,63}[.]?)* -- Match 1 to max 63 of these characters
		//                              Followed by exactly 0 or no periods
		//                              Match the above combination zero or more times
		// [a-zA-Z]{2,63}            -- Match at least two, max 63 of these characters
		// $                         -- End of string
		if match, err := regexp.MatchString(`^[a-zA-Z0-9]+([a-zA-Z0-9-]{1,63}[.]?)*[a-zA-Z]{2,63}$`, host); err != nil {
			panic("invalid regular expression for hostname")
		} else if !match {
			panic(fmt.Errorf("invalid hostname %s in endpoint config", host))
		}

		port := parts[1]
		if port == "" {
			port = fmt.Sprintf("%d", defaultEndpointPort)
		} else if portNum, err := strconv.Atoi(port); err != nil {
			panic(fmt.Errorf("port %s in endpoint config is NaN", port))
		} else if portNum < 0 || portNum > maxPort {
			panic(fmt.Errorf("port %s not in valid RFC range (0-%d)", port, maxPort))
		}
		return fmt.Sprintf("%s:%s", host, port)
	}
	return fmt.Sprintf("%s:%d", endpoint, defaultEndpointPort)
}

func NewWSConfig() (config WSConfig) {
	config.PublicKeyPath = os.Getenv(publicEnv)
	if config.PublicKeyPath == "" {
		config.PublicKeyPath = defaultPublicPath
	}

	config.PrivateKeyPath = os.Getenv(privateKeyEnv)
	if config.PrivateKeyPath == "" {
		config.PrivateKeyPath = defaultPrivatePath
	}

	config.Endpoint = formatEndpoint(os.Getenv(endpointEnv))
	value := os.Getenv(allowedOriginsEnv)
	if strings.TrimSpace(value) != "" {
		config.AllowedOrigins = getOriginsAsSlice(value)
	}

	return config
}

// getOriginsAsSlice turns the given value of and env var into a slice of strings.
// It trims spaces from each value and ignores empty values.
func getOriginsAsSlice(value string) []string {
	if value == "*" {
		return []string{"*"}
	}

	// Split by commas and trim spaces
	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

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
	defaultPublicPath   = "/secrets/paas/publicKey"
	endpointEnv         = "PAAS_ENDPOINT"
	defaultEndpointPort = 8080
)

type WSConfig struct {
	PublicKeyPath string
	Endpoint      string
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

	return config

}

package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	privateEnv         = "PAAS_PRIVATE_KEY_PATH"
	defaultPrivatePath = "/secrets/paaswebservice/privatekey"
	publicEnv          = "PAAS_PUBLIC_KEY_PATH"
	defaultPublicPath  = "/secrets/paaswebservice/publickey"
	publicEndpoint     = "PAAS_ENDPOINT"
	defaultEndpoint    = "localhost:8080"
)

type WSConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
	Endpoint       string
}

func NewWSConfig() WSConfig {
	var config WSConfig
	config.PrivateKeyPath = os.Getenv(privateEnv)
	if config.PrivateKeyPath == "" {
		config.PrivateKeyPath = defaultPrivatePath
	}
	config.PublicKeyPath = os.Getenv(publicEnv)
	if config.PublicKeyPath == "" {
		config.PublicKeyPath = defaultPublicPath
	}
	config.Endpoint = os.Getenv(publicEnv)
	if config.Endpoint == "" {
		config.Endpoint = defaultPublicPath
	} else if strings.Contains(config.Endpoint, ":") {
		parts := strings.Split(config.Endpoint, ":")
		host := parts[0]
		if host == "" {
			host = "localhost"
		} else if len(host) > 63 {
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
	return config

}

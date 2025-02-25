/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

// Package validate provides various high-level convenience functions to help
// validate components within the Paas Operator such as fields of a PaasConfig
// resource.
package validate

import (
	"fmt"
	"net"
	"regexp"
)

// StringIsRegex checks if a given string is a compilable regex.
func StringIsRegex(regex string) (bool, error) {
	if _, err := regexp.Compile(regex); err != nil {
		return false, fmt.Errorf("uncompilable regular expression")
	}

	return true, nil
}

// Hostname checks if a given string is either a valid IP or valid hostname.
func Hostname(hostname string) (bool, error) {
	// checks if the input string is a valid hostname according to RFC 1035
	hostnameRegex := `^([a-zA-Z0-9][a-zA-Z0-9\-]{0,62}\.)+[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(hostnameRegex, hostname)

	// ParseIP checks if the input string is a valid IPv4 or IPv6 address
	if net.ParseIP(hostname) == nil && !match {
		return false, fmt.Errorf("invalid host name / ip address")
	}

	return true, nil
}

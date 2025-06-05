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
	"errors"
	"regexp"
)

// StringIsRegex checks if a given string is a compilable regex.
func StringIsRegex(regex string) (bool, error) {
	if _, err := regexp.Compile(regex); err != nil {
		return false, errors.New("uncompilable regular expression")
	}

	return true, nil
}

/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/sirupsen/logrus"
)

// CheckPaas determines whether a Paas can be decrypted using the provided crypt
// it returns an error containing which secrets cannot be decrypted if any
func CheckPaas(cryptObj *crypt.Crypt, paas *v1alpha1.Paas) error {
	var allErrors []string
	for key, secret := range paas.Spec.SSHSecrets {
		decrypted, err := cryptObj.Decrypt(secret)
		if err != nil {
			errMessage := fmt.Errorf("%s: .spec.sshSecrets[%s], error: %w", paas.Name, key, err)
			logrus.Error(errMessage)
			allErrors = append(allErrors, errMessage.Error())
		} else {
			logrus.Infof(
				"%s: .spec.sshSecrets[%s], checksum: %s, len %d",
				paas.Name,
				key,
				hashData(decrypted),
				len(decrypted),
			)
		}
	}

	for capName, capability := range paas.Spec.Capabilities {
		logrus.Debugf("capability name: %s", capName)
		for key, secret := range capability.GetSSHSecrets() {
			decrypted, err := cryptObj.Decrypt(secret)
			if err != nil {
				errMessage := fmt.Errorf(
					"%s: .spec.capabilities[%s].sshSecrets[%s], error: %w",
					paas.Name,
					capName,
					key,
					err,
				)
				logrus.Error(errMessage)
				allErrors = append(allErrors, errMessage.Error())
			} else {
				logrus.Infof("%s: .spec.capabilities[%s].sshSecrets[%s], checksum: %s, len %d.",
					paas.Name,
					capName,
					key,
					hashData(decrypted),
					len(decrypted),
				)
			}
		}
	}
	if len(allErrors) > 0 {
		errorString := strings.Join(allErrors, " , ")
		return errors.New(errorString)
	}
	return nil
}

func hashData(original []byte) string {
	sum := sha512.Sum512(original)
	return hex.EncodeToString(sum[:])
}

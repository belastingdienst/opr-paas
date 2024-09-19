/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/belastingdienst/opr-paas/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func checkPaasFiles(privateKeyFiles string, files []string) error {
	var errNum int

	srcCrypt, err := crypt.NewCrypt([]string{privateKeyFiles}, "", "checkPaas")
	if err != nil {
		return err
	}
	for _, fileName := range files {
		// Read paas from file
		paas, _, err := readPaasFile(fileName)
		if err != nil {
			return fmt.Errorf("could not read file %s: %s", fileName, err.Error())
		}

		checkPaasErr := CheckPaas(srcCrypt, paas, fileName)
		if checkPaasErr != nil {
			errNum++
		}
	}

	errMsg := fmt.Errorf("finished with %d errors", errNum)
	if errNum > 0 {
		logrus.Error(errMsg)
		return errMsg
	}

	logrus.Info(errMsg)

	return nil
}

// CheckPaas determines whether a Paas can be decrypted using the provided crypt
// it returns an error containing which secrets cannot be decrypted if any
func CheckPaas(crypt *crypt.Crypt, paas *v1alpha1.Paas, fileName string) error {
	var allErrors []string
	for key, secret := range paas.Spec.SshSecrets {
		decrypted, err := crypt.Decrypt(secret)
		if err != nil {
			errMessage := fmt.Errorf("%s: { .spec.sshSecrets[%s] } > { error: %w }", fileName, key, err)
			logrus.Error(errMessage)
			allErrors = append(allErrors, errMessage.Error())
		} else {
			logrus.Infof("%s: { .spec.sshSecrets[%s] } > { checksum: %s, len %d }", fileName, key, hashData(decrypted), len(decrypted))
		}
	}

	for capName, capability := range paas.Spec.Capabilities.AsMap() {
		logrus.Debugf("capability name: %s", capability.CapabilityName())
		for key, secret := range capability.GetSshSecrets() {
			decrypted, err := crypt.Decrypt(secret)
			if err != nil {
				errMessage := fmt.Errorf("%s: { .spec.capabilities[%s].sshSecrets[%s] } > { error: %w }", fileName, capName, key, err)
				logrus.Error(errMessage)
				allErrors = append(allErrors, errMessage.Error())
			} else {
				logrus.Infof("%s: { .spec.capabilities[%s].sshSecrets[%s] } > { checksum: %s, len %d }", fileName, capName, key, hashData(decrypted), len(decrypted))
			}
		}
	}
	if len(allErrors) > 0 {
		errorString := strings.Join(allErrors, " , ")
		return errors.New(errorString)
	}
	return nil
}

func checkPaasCmd() *cobra.Command {
	var privateKeyFiles string

	cmd := &cobra.Command{
		Use:   "check-paas [command options]",
		Short: "check secrets in paas yaml files",
		Long:  `check-paas can parse yaml/json files with paas objects, decrypt the sshSecrets and display length and checksum.`,
		RunE: func(command *cobra.Command, args []string) error {
			if debug {
				logrus.SetLevel(logrus.DebugLevel)
			}

			files, err := utils.PathToFileList(args)
			if err != nil {
				return err
			}

			return checkPaasFiles(privateKeyFiles, files)
		},
		Args:    cobra.MinimumNArgs(1),
		Example: `crypttool check-paas --privateKeyFiles "/tmp/priv" [file or dir] ([file or dir]...)`,
	}

	flags := cmd.Flags()
	flags.StringVar(&privateKeyFiles, "privateKeyFiles", "", "The file or folder containing the private key(s)")

	if err := viper.BindPFlag("privateKeyFiles", flags.Lookup("privateKeyFiles")); err != nil {
		logrus.Errorf("key binding for private key failed: %v", err)
	}
	if err := viper.BindEnv("privateKeyFiles", "PAAS_PRIVATE_KEY_PATH"); err != nil {
		logrus.Errorf("key binding for PAAS_PRIVATE_KEY_PATH failed: %v", err)
	}

	return cmd
}

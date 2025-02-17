/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"fmt"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func encryptCmd() *cobra.Command {
	var publicKeyFile string
	var dataFile string
	var paasName string

	cmd := &cobra.Command{
		Use:   "encrypt [command options]",
		Short: "encrypt using public key and print results",
		Long:  `encrypt using public key and print results`,
		RunE: func(command *cobra.Command, args []string) error {
			if paasName == "" {
				return fmt.Errorf("a paas must be set with eith --paas or environment variabele PAAS_NAME")
			}
			if dataFile == "" {
				return crypt.EncryptFromStdin(publicKeyFile, paasName)
			} else {
				return crypt.EncryptFile(publicKeyFile, paasName, dataFile)
			}
		},
		Example: `crypttool encrypt --publicKeyFile "/tmp/pub" --dataFile "/tmp/decrypted" --paas my-paas`,
	}

	flags := cmd.Flags()
	flags.StringVar(&publicKeyFile, publicKeyFile, "", "The file to read the public key from")
	flags.StringVar(&dataFile, argNameDataFileKey, "", "The file to read the data to be encrypted from")
	flags.StringVar(&paasName, argNamePaas, "", "The paas this data is to be encrypted for")

	if err := viper.BindPFlag(publicKeyFile, flags.Lookup(argNamePublicKeyFile)); err != nil {
		logrus.Errorf("error binding public key file: %v", err)
	}
	if err := viper.BindPFlag(argNameDataFileKey, flags.Lookup(argNameDataFileKey)); err != nil {
		logrus.Errorf("error binding data file: %v", err)
	}
	if err := viper.BindPFlag(argNamePaas, flags.Lookup(argNamePaas)); err != nil {
		logrus.Errorf("error binding paas key: %v", err)
	}
	if err := viper.BindEnv(argNamePublicKeyFile, "PAAS_PUBLIC_KEY_PATH"); err != nil {
		logrus.Errorf("error binding paas public key: %v", err)
	}
	if err := viper.BindEnv(argNameDataFileKey, "PAAS_INPUT_FILE"); err != nil {
		logrus.Errorf("error binding paas data file key: %v", err)
	}
	if err := viper.BindEnv(argNamePaas, "PAAS_NAME"); err != nil {
		logrus.Errorf("error binding paas name: %v", err)
	}

	return cmd
}

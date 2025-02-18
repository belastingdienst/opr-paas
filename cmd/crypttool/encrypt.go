/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"errors"

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
		RunE: func(_ *cobra.Command, _ []string) error {
			if paasName == "" {
				return errors.New("a paas must be set with with --paas or environment variabele PAAS_NAME")
			}
			if dataFile == "" {
				return crypt.EncryptFromStdin(publicKeyFile, paasName)
			}
			return crypt.EncryptFile(publicKeyFile, paasName, dataFile)
		},
		Example: `crypttool encrypt --publicKeyFile "/tmp/pub" --dataFile "/tmp/decrypted" --paas my-paas`,
	}

	flags := cmd.Flags()
	flags.StringVar(&publicKeyFile, "publicKeyFile", "", "The file to read the public key from")
	flags.StringVar(&dataFile, "dataFile", "", "The file to read the data to be encrypted from")
	flags.StringVar(&paasName, "paas", "", "The paas this data is to be encrypted for")

	if err := viper.BindPFlag("publicKeyFile", flags.Lookup("publicKeyFile")); err != nil {
		logrus.Errorf("error binding public key file: %v", err)
	}
	if err := viper.BindPFlag("dataFile", flags.Lookup("dataFile")); err != nil {
		logrus.Errorf("error binding data file: %v", err)
	}
	if err := viper.BindPFlag("paas", flags.Lookup("paas")); err != nil {
		logrus.Errorf("error binding paas key: %v", err)
	}
	if err := viper.BindEnv("publicKeyFile", "PAAS_PUBLIC_KEY_PATH"); err != nil {
		logrus.Errorf("error binding paas public key: %v", err)
	}
	if err := viper.BindEnv("dataFile", "PAAS_INPUT_FILE"); err != nil {
		logrus.Errorf("error binding paas data file key: %v", err)
	}
	if err := viper.BindEnv("paas", "PAAS_NAME"); err != nil {
		logrus.Errorf("error binding paas name: %v", err)
	}

	return cmd
}

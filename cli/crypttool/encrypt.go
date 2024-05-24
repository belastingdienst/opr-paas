/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"fmt"

	"github.com/belastingdienst/opr-paas/internal/crypt"
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
	flags.StringVar(&publicKeyFile, "publicKeyFile", "", "The file to read the public key from")
	flags.StringVar(&dataFile, "dataFile", "", "The file to read the data to be encrypted from")
	flags.StringVar(&paasName, "paas", "", "The paas this data is to be encrypted for")

	viper.BindPFlag("publicKeyFile", flags.Lookup("publicKeyFile"))
	viper.BindPFlag("dataFile", flags.Lookup("dataFile"))
	viper.BindPFlag("paas", flags.Lookup("paas"))

	viper.BindEnv("publicKeyFile", "PAAS_PUBLIC_KEY_PATH")
	viper.BindEnv("dataFile", "PAAS_INPUT_FILE")
	viper.BindEnv("paas", "PAAS_NAME")

	return cmd
}

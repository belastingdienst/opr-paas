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

func decryptCmd() *cobra.Command {
	var privateKeyFiles string
	var paasName string

	cmd := &cobra.Command{
		Use:   "decrypt [command options]",
		Short: "decrypt using private key and print results",
		Long:  `decrypt using private key and print results`,
		RunE: func(command *cobra.Command, args []string) error {
			if paasName == "" {
				return fmt.Errorf("a paas must be set with eith --paas or environment variabele PAAS_NAME")
			}
			return crypt.DecryptFromStdin([]string{privateKeyFiles}, paasName)
		},
		Example: `crypttool decrypt --privateKeyFiles "/tmp/priv" --paas my-paas`,
	}

	flags := cmd.Flags()
	flags.StringVar(&privateKeyFiles, argNamePrivateKeyFiles, "", "The file to read the private key from")
	flags.StringVar(&paasName, argNamePaas, "", "The paas this data is to be encrypted for")

	if err := viper.BindPFlag(argNamePrivateKeyFiles, flags.Lookup(argNamePrivateKeyFiles)); err != nil {
		logrus.Errorf("error binding private keys: %v", err)
	}
	if err := viper.BindPFlag(argNamePaas, flags.Lookup(argNamePaas)); err != nil {
		logrus.Errorf("error binding paas key: %v", err)
	}
	if err := viper.BindEnv(argNamePrivateKeyFiles, "PAAS_PRIVATE_KEY_PATH"); err != nil {
		logrus.Errorf("error binding paas private keys: %v", err)
	}
	if err := viper.BindEnv(argNamePaas, "PAAS_NAME"); err != nil {
		logrus.Errorf("error binding PAAS_NAME: %v", err)
	}

	return cmd
}

package main

import (
	"fmt"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func decryptCmd() *cobra.Command {
	var privateKeyFile string
	var paasName string

	cmd := &cobra.Command{
		Use:   "decrypt [command options]",
		Short: "decrypt using private key and print results",
		Long:  `decrypt using private key and print results`,
		RunE: func(command *cobra.Command, args []string) error {
			if paasName == "" {
				return fmt.Errorf("a paas must be set with eith --paas or environment variabele PAAS_NAME")
			}
			return crypt.DecryptFromStdin(privateKeyFile, paasName)
		},
		Example: `crypttool decrypt --privateKeyFile "/tmp/priv" --paas my-paas`,
	}

	flags := cmd.Flags()
	flags.StringVar(&privateKeyFile, "privateKeyFile", "", "The file to read the private key from")
	viper.BindPFlag("privateKeyFile", flags.Lookup("privateKeyFile"))
	viper.BindEnv("privateKeyFile", "PAAS_PRIVATE_KEY_PATH")
	flags.StringVar(&paasName, "paas", "", "The paas this data is to be encrypted for")
	viper.BindPFlag("paas", flags.Lookup("paas"))
	viper.BindEnv("paas", "PAAS_NAME")

	return cmd
}

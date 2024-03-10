package main

import (
	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func generateCmd() *cobra.Command {
	var publicKeyFile string
	var privateKeyFile string

	cmd := &cobra.Command{
		Use:   "generate [command options]",
		Short: "generate a new private and public key and store them in files",
		Long:  `generate a new private and public key and store them in files`,
		RunE: func(command *cobra.Command, args []string) error {
			return crypt.GenerateKeyPair(privateKeyFile, publicKeyFile)
		},
		Example: `crypttool generate --publicKeyFile "/tmp/pub" --privateKeyFile "/tmp/priv"`,
	}

	flags := cmd.Flags()
	flags.StringVar(&publicKeyFile, "publicKeyFile", "", "The file to write the public key to")
	viper.BindPFlag("publicKeyFile", flags.Lookup("publicKeyFile"))
	viper.BindEnv("publicKeyFile", "PAAS_PUBLIC_KEY_PATH")
	flags.StringVar(&privateKeyFile, "privateKeyFile", "", "The file to write the private key to")
	viper.BindPFlag("privateKeyFile", flags.Lookup("privateKeyFile"))
	viper.BindEnv("privateKeyFile", "PAAS_PRIVATE_KEY_PATH")

	return cmd
}

package main

import (
	"fmt"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/belastingdienst/opr-paas/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func reencryptFiles(privateKeyFiles string, publicKeyFile string, outputFormat string, files []string) error {
	for _, fileName := range files {
		// Read paas from file
		paas, format, err := readPaasFile(fileName)
		if err != nil {
			return fmt.Errorf("could not read file")
		}

		paasName := paas.ObjectMeta.Name
		srcCrypt, err := crypt.NewCrypt([]string{privateKeyFiles}, "", paasName)
		if err != nil {
			return err
		}

		dstCrypt, err := crypt.NewCrypt([]string{}, publicKeyFile, paasName)
		if err != nil {
			return nil
		}

		for key, secret := range paas.Spec.SshSecrets {
			decrypted, err := srcCrypt.Decrypt(secret)
			if err != nil {
				return fmt.Errorf("failed to decrypt %s.spec.sshSecrets[%s] in %s: %e", paasName, key, fileName, err)
			}

			reencrypted, err := dstCrypt.Encrypt(decrypted)
			if err != nil {
				return fmt.Errorf("failed to reecrypt %s.spec.sshSecrets[%s] in %s: %e", paasName, key, fileName, err)
			}

			logrus.Debugf("succesfully reencrypted %s.spec.sshSecrets[%s] in file %s", paasName, key, fileName)
			logrus.Debugf("decrypted: {checksum: %s, len: %d}", hashData(decrypted), len(decrypted))
			logrus.Debugf("reencrypted: {checksum: %s, len: %d}", hashData([]byte(reencrypted)), len(reencrypted))
			paas.Spec.SshSecrets[key] = reencrypted
		}

		for capName, cap := range paas.Spec.Capabilities.AsMap() {
			for key, secret := range cap.GetSshSecrets() {
				decrypted, err := srcCrypt.Decrypt(secret)
				if err != nil {
					return fmt.Errorf("failed to decrypt %s.spec.capabilities.%s.sshSecrets[%s] in %s: %e", paasName, capName, key, fileName, err)
				}

				reencrypted, err := dstCrypt.Encrypt(decrypted)
				if err != nil {
					return fmt.Errorf("failed to reecrypt %s.spec.capabilities.%s.sshSecrets[%s] in %s: %e", paasName, capName, key, fileName, err)
				}

				logrus.Debugf("succesfully reencrypted %s.spec.capabilities[%s].sshSecrets[%s] in file %s", paasName, capName, key, fileName)
				logrus.Debugf("decrypted: {checksum: %s, len: %d}", hashData(decrypted), len(decrypted))
				logrus.Debugf("reencrypted: {checksum: %s, len: %d}", hashData([]byte(reencrypted)), len(reencrypted))
				cap.SetSshSecret(key, reencrypted)
			}
		}

		// Write paas to file
		if outputFormat != "auto" {
			format = outputFormat
		}

		switch format {
		case "json":
			if err = writePaasJsonFile(paas, fileName); err != nil {
				return err
			}
		case "yaml":
			if err = writePaasYamlFile(paas, fileName); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid output format: %s", format)
		}

	}
	return nil
}

func reencryptCmd() *cobra.Command {
	var privateKeyFiles string
	var publicKeyFile string
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "reencrypt [command options]",
		Short: "reencrypt using old private key to decrypt and new public key to encrypt",
		Long: `reencrypt can parse yaml/json files with paas objects, decrypt the sshSecrets with the previous private key,
reencrypt with the new public key and write back the paas to the file in either yaml or json format.`,
		RunE: func(command *cobra.Command, args []string) error {
			if debug {
				logrus.SetLevel(logrus.DebugLevel)
			}
			if files, err := utils.PathToFileList(args); err != nil {
				return err
			} else {
				return reencryptFiles(privateKeyFiles, publicKeyFile, outputFormat, files)
			}
		},
		Args:    cobra.MinimumNArgs(1),
		Example: `crypttool reencrypt --privateKeyFiles "/tmp/priv" --publicKeyFile "/tmp/pub" [file or dir] ([file or dir]...)`,
	}

	flags := cmd.Flags()
	flags.StringVar(&privateKeyFiles, "privateKeyFiles", "", "The file to read the private key from")
	viper.BindPFlag("privateKeyFiles", flags.Lookup("privateKeyFiles"))
	viper.BindEnv("privateKeyFiles", "PAAS_PRIVATE_KEY_PATH")
	flags.StringVar(&publicKeyFile, "publicKeyFile", "", "The file to read the public key from")
	viper.BindPFlag("publicKeyFile", flags.Lookup("publicKeyFile"))
	viper.BindEnv("publicKeyFile", "PAAS_PUBLIC_KEY_PATH")
	flags.StringVar(&outputFormat, "outputFormat", "auto", "The outputformat for writing a paas, either yaml, json, or auto (which will revert to same format as input)")
	viper.BindPFlag("outputFormat", flags.Lookup("outputFormat"))
	viper.BindEnv("outputFormat", "PAAS_OUTPUT_FORMAT")
	return cmd
}

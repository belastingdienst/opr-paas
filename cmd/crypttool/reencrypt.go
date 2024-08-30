/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/belastingdienst/opr-paas/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// reencryptSecret decrypts, then re-encrypts a given secret using given src and
// destination crypt.Crypt instances.
func reencryptSecret(srcCrypt *crypt.Crypt, dstCrypt *crypt.Crypt, secret string) (string, error) {
	decrypted, err := srcCrypt.Decrypt(secret)
	if err != nil {
		return "", err
	}
	logrus.Debugf("decrypted: {checksum: %s, len: %d}", hashData(decrypted), len(decrypted))

	reencrypted, err := dstCrypt.Encrypt(decrypted)
	if err != nil {
		return "", err
	}
	logrus.Debugf("reencrypted: {checksum: %s, len: %d}", hashData([]byte(reencrypted)), len(reencrypted))

	return reencrypted, nil
}

func reencryptFiles(privateKeyFiles string, publicKeyFile string, outputFormat string, files []string) error {
	for _, fileName := range files {
		// Read paas as String to preserve format
		paasAsBytes, err := os.ReadFile(fileName)
		paasAsString := string(paasAsBytes)
		if err != nil {
			return fmt.Errorf("could not read file into string")
		}

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
			reencrypted, err := reencryptSecret(srcCrypt, dstCrypt, secret)
			if err != nil {
				return fmt.Errorf("failed to decrypt/reencrypt %s.spec.sshSecrets[%s] in %s: %w", paasName, key, fileName, err)
			}

			paas.Spec.SshSecrets[key] = reencrypted
			// Use replaceAll as same secret can occur multiple times
			paasAsString = strings.ReplaceAll(paasAsString, secret, reencrypted)
			logrus.Debugf("successfully reencrypted %s.spec.sshSecrets[%s] in file %s", paasName, key, fileName)
		}

		for capName, cap := range paas.Spec.Capabilities.AsMap() {
			for key, secret := range cap.GetSshSecrets() {
				reencrypted, err := reencryptSecret(srcCrypt, dstCrypt, secret)
				if err != nil {
					return fmt.Errorf("failed to decrypt/reencrypt %s.spec.capabilities.%s.sshSecrets[%s] in %s: %w", paasName, capName, key, fileName, err)
				}

				cap.SetSshSecret(key, reencrypted)
				// Use replaceAll as same secret can occur multiple times
				paasAsString = strings.ReplaceAll(paasAsString, secret, reencrypted)
				logrus.Debugf("successfully reencrypted %s.spec.capabilities[%s].sshSecrets[%s] in file %s", paasName, capName, key, fileName)
			}
		}

		// Write paas to file
		if outputFormat != "auto" {
			format = outputFormat
		}

		if outputFormat == "preserved" {
			err := writeFile([]byte(paasAsString), fileName)
			if err != nil {
				return err
			}
		} else {
			err := writeFormattedFile(paas, fileName, format)
			if err != nil {
				return err
			}
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
	flags.StringVar(&publicKeyFile, "publicKeyFile", "", "The file to read the public key from")
	flags.StringVar(&outputFormat, "outputFormat", "auto", "The outputformat for writing a paas, either yaml (machine formatted), json (machine formatted), auto (which will use input format as output, machine formatted) or preserved (which will use the input format and preserve the original syntax including for example comments) ")

	if err := viper.BindPFlag("privateKeyFiles", flags.Lookup("privateKeyFiles")); err != nil {
		logrus.Errorf("key binding for privatekeyfiles failed: %v", err)
	}
	if err := viper.BindPFlag("publicKeyFile", flags.Lookup("publicKeyFile")); err != nil {
		logrus.Errorf("key binding for publickeyfile failed: %v", err)
	}
	if err := viper.BindPFlag("outputFormat", flags.Lookup("outputFormat")); err != nil {
		logrus.Errorf("key binding at output step failed: %v", err)
	}
	if err := viper.BindEnv("privateKeyFiles", "PAAS_PRIVATE_KEY_PATH"); err != nil {
		logrus.Errorf("private key to env var binding failed: %v", err)
	}
	if err := viper.BindEnv("publicKeyFile", "PAAS_PUBLIC_KEY_PATH"); err != nil {
		logrus.Errorf("public key to env var binding failed: %v", err)
	}
	if err := viper.BindEnv("outputFormat", "PAAS_OUTPUT_FORMAT"); err != nil {
		logrus.Errorf("key binding at output step failed: %v", err)
	}

	return cmd
}

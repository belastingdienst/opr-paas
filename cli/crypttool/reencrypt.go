package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type InvalidPaasFile struct {
	File string
}

func (ip *InvalidPaasFile) Error() string {
	return fmt.Sprintf("file '%s' does not contain a valid paas", ip.File)
}

func readPaasFile(file string) (*v1alpha1.Paas, string, error) {

	var paas v1alpha1.Paas

	buffer, err := os.ReadFile(file)
	if err != nil {
		return nil, "unknown", err
	}

	err = json.Unmarshal(buffer, &paas)
	if err == nil {
		return &paas, "json", nil
	}

	err = yaml.Unmarshal(buffer, &paas)
	if err == nil {
		return &paas, "yaml", nil
	}

	return nil, "unknown", &InvalidPaasFile{File: file}
}

func writePaasJsonFile(paas *v1alpha1.Paas, path string) error {

	if buffer, err := json.Marshal(&paas); err != nil {
		return err
	} else if file, err := os.Create(path); err != nil {
		return err
	} else if _, err := file.Write(buffer); err != nil {
		return err
	} else {
		log.Printf("File '%s' succefully updated as json", path)
		return nil
	}
}

func writePaasYamlFile(paas *v1alpha1.Paas, path string) error {

	if buffer, err := yaml.Marshal(&paas); err != nil {
		return err
	} else if file, err := os.Create(path); err != nil {
		return err
	} else if _, err := file.Write(buffer); err != nil {
		return err
	} else {
		log.Printf("File '%s' succefully updated as yaml", path)
		return nil
	}
}

func processFiles(privateKeyFile string, publicKeyFile string, outputFormat string, files []string) error {
	for _, fileName := range files {
		// Read paas from file
		if paas, format, err := readPaasFile(fileName); err != nil {
			return fmt.Errorf("could not read file")
		} else {
			paasName := paas.ObjectMeta.Name
			srcCrypt := crypt.NewCrypt(privateKeyFile, "", paasName)
			dstCrypt := crypt.NewCrypt("", publicKeyFile, paasName)
			for key, secret := range paas.Spec.SshSecrets {
				if decrypted, err := srcCrypt.Decrypt(secret); err != nil {
					return fmt.Errorf("failed to decrypt %s.spec.sshSecrets[%s] in %s: %e", paasName, key, fileName, err)
				} else if reencrypted, err := dstCrypt.Encrypt(decrypted); err != nil {
					return fmt.Errorf("failed to reecrypt %s.spec.sshSecrets[%s] in %s: %e", paasName, key, fileName, err)
				} else {
					paas.Spec.SshSecrets[key] = reencrypted
				}
			}
			for capName, cap := range paas.Spec.Capabilities.AsMap() {
				for key, secret := range (*cap).GetSshSecrets() {
					if decrypted, err := srcCrypt.Decrypt(secret); err != nil {
						return fmt.Errorf("failed to decrypt %s.spec.capabilities.%s.sshSecrets[%s] in %s: %e", paasName, capName, key, fileName, err)
					} else if reencrypted, err := dstCrypt.Encrypt(decrypted); err != nil {
						return fmt.Errorf("failed to reecrypt %s.spec.capabilities.%s.sshSecrets[%s] in %s: %e", paasName, capName, key, fileName, err)
					} else {
						(*cap).SetSshSecret(key, reencrypted)
					}
				}
			}
			// Write paas to file
			if outputFormat != "auto" {
				format = outputFormat
			}
			if format == "json" {
				if err = writePaasJsonFile(paas, fileName); err != nil {
					return err
				}
			} else if format == "yaml" {
				if err = writePaasYamlFile(paas, fileName); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("invalid output format: %s", format)
			}
		}
	}
	return nil
}

func reencryptCmd() *cobra.Command {
	var privateKeyFile string
	var publicKeyFile string
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "reencrypt [command options]",
		Short: "reencrypt using old private key to decrypt and new public key to encrypt",
		Long: `reencrypt can parse yaml/json files with paas objects, decrypt the sshSecrets with the previous private key,
reencrypt with the new public key and write back the paas to the file in either yaml or json format.`,
		RunE: func(command *cobra.Command, args []string) error {
			if files, err := pathToFileList(args); err != nil {
				return err
			} else {
				return processFiles(privateKeyFile, publicKeyFile, outputFormat, files)
			}
		},
		Args:    cobra.MinimumNArgs(1),
		Example: `crypttool reencrypt --privateKeyFile "/tmp/priv" --publicKeyFile "/tmp/pub" [file or dir] ([file or dir]...)`,
	}

	flags := cmd.Flags()
	flags.StringVar(&privateKeyFile, "privateKeyFile", "", "The file to read the private key from")
	viper.BindPFlag("privateKeyFile", flags.Lookup("privateKeyFile"))
	viper.BindEnv("privateKeyFile", "PAAS_PRIVATE_KEY_PATH")
	flags.StringVar(&publicKeyFile, "publicKeyFile", "", "The file to read the public key from")
	viper.BindPFlag("publicKeyFile", flags.Lookup("publicKeyFile"))
	viper.BindEnv("publicKeyFile", "PAAS_PUBLIC_KEY_PATH")
	flags.StringVar(&outputFormat, "outputFormat", "auto", "The outputformat for writing a paas, either yaml, json, or auto (which will revert to same format as input)")
	viper.BindPFlag("outputFormat", flags.Lookup("outputFormat"))
	viper.BindEnv("outputFormat", "PAAS_OUTPUT_FORMAT")
	return cmd
}

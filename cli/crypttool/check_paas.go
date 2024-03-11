package main

import (
	"fmt"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func checkPaasFiles(privateKeyFile string, files []string) error {
	var errNum int
	for _, fileName := range files {
		// Read paas from file
		if paas, _, err := readPaasFile(fileName); err != nil {
			return fmt.Errorf("could not read file %s: %s", fileName, err.Error())
		} else {
			paasName := paas.ObjectMeta.Name
			srcCrypt := crypt.NewCrypt(privateKeyFile, "", paasName)
			for key, secret := range paas.Spec.SshSecrets {
				if decrypted, err := srcCrypt.Decrypt(secret); err != nil {
					errNum += 1
					logrus.Errorf("%s: { .spec.sshSecrets[%s] } > { error: %e }", fileName, key, err)
				} else {
					logrus.Infof("%s: { .spec.sshSecrets[%s] } > { checksum: %s, len %d }", fileName, key, hashData(decrypted), len(decrypted))
				}
			}
			for capName, cap := range paas.Spec.Capabilities.AsMap() {
				logrus.Debugf("cap name: %s", cap.CapabilityName())
				for key, secret := range cap.GetSshSecrets() {
					if decrypted, err := srcCrypt.Decrypt(secret); err != nil {
						logrus.Errorf("%s: { .spec.capabilities[%s].sshSecrets[%s] } > { error: %e }", fileName, capName, key, err)
						errNum += 1
					} else {
						logrus.Infof("%s: { .spec.capabilities[%s].sshSecrets[%s] } > { checksum: %s, len %d }", fileName, capName, key, hashData(decrypted), len(decrypted))
					}
				}
			}
		}
	}
	errMsg := fmt.Sprintf("Finished with %d errors", errNum)
	if errNum > 0 {
		logrus.Error(errMsg)
		return fmt.Errorf(errMsg)
	}
	logrus.Infof(errMsg)
	return nil
}

func checkPaasCmd() *cobra.Command {
	var privateKeyFile string

	cmd := &cobra.Command{
		Use:   "check-paas [command options]",
		Short: "check secrets in paas yaml files",
		Long:  `check-paas can parse yaml/json files with paas objects, decrypt the sshSecrets and display length and checksum.`,
		RunE: func(command *cobra.Command, args []string) error {
			if debug {
				logrus.SetLevel(logrus.DebugLevel)
			}
			if files, err := pathToFileList(args); err != nil {
				return err
			} else {
				return checkPaasFiles(privateKeyFile, files)
			}
		},
		Args:    cobra.MinimumNArgs(1),
		Example: `crypttool check-paas --privateKeyFile "/tmp/priv" [file or dir] ([file or dir]...)`,
	}

	flags := cmd.Flags()
	flags.StringVar(&privateKeyFile, "privateKeyFile", "", "The file to read the private key from")
	viper.BindPFlag("privateKeyFile", flags.Lookup("privateKeyFile"))
	viper.BindEnv("privateKeyFile", "PAAS_PRIVATE_KEY_PATH")
	return cmd
}

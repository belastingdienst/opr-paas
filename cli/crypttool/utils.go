package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

// TODO: utils.go should not exist, clean it up

type InvalidPaasFile struct {
	File string
}

func (ip *InvalidPaasFile) Error() string {
	return fmt.Sprintf("file '%s' does not contain a valid paas", ip.File)
}

func readPaasFile(file string) (*v1alpha1.Paas, string, error) {
	var paas v1alpha1.Paas

	logrus.Debugf("parsing %s", file)
	buffer, err := os.ReadFile(file)
	if err != nil {
		logrus.Debugf("could not read %s: %e", file, err)
		return nil, "unknown", err
	}

	err = json.Unmarshal(buffer, &paas)
	if err == nil {
		return &paas, "json", nil
	}
	logrus.Debugf("could not parse %s as json: %e", file, err)

	err = yaml.Unmarshal(buffer, &paas)
	if err == nil {
		return &paas, "yaml", nil
	}
	logrus.Debugf("could not parse %s as yaml: %e", file, err)

	return nil, "unknown", &InvalidPaasFile{File: file}
}

func writeFile(buffer []byte, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	if _, err := file.Write(buffer); err != nil {
		return err
	}

	log.Printf("file '%s' succesfully updated", path)
	return nil
}

func writePaasJsonFile(paas *v1alpha1.Paas, path string) error {
	buffer, err := json.Marshal(&paas)
	if err != nil {
		return err
	}

	return writeFile(buffer, path)
}

func writePaasYamlFile(paas *v1alpha1.Paas, path string) error {
	buffer, err := yaml.Marshal(&paas)
	if err != nil {
		return err
	}

	return writeFile(buffer, path)
}

func hashData(original []byte) string {
	sum := sha512.Sum512(original)
	return hex.EncodeToString(sum[:])
}

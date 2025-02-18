/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

type InvalidPaasFileFormat struct {
	File string
}

func (ip *InvalidPaasFileFormat) Error() string {
	return fmt.Sprintf("file '%s' is not in a supported file format", ip.File)
}

func readPaasFile(file string) (*v1alpha1.Paas, string, error) {
	var paas v1alpha1.Paas

	logrus.Debugf("parsing %s", file)
	buffer, err := os.ReadFile(file)
	if err != nil {
		logrus.Debugf("could not read %s: %e", file, err)
		return nil, "unable to read paas configuration file", err
	}

	if len(buffer) == 0 {
		return nil, "", errors.New("empty paas configuration file")
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

	return nil, "", &InvalidPaasFileFormat{File: file}
}

func writeFile(buffer []byte, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	if _, err := file.Write(buffer); err != nil {
		return err
	}

	logrus.Infof("file '%s' successfully updated", path)
	return nil
}

func writeFormattedFile(paas *v1alpha1.Paas, path string, format string) error {
	var buffer []byte
	var err error

	switch format {
	default:
		return fmt.Errorf("invalid output format: %s", format)
	case "json":
		buffer, err = json.Marshal(&paas)
	case "yaml":
		buffer, err = yaml.Marshal(&paas)
	}

	if err != nil {
		return err
	}

	return writeFile(buffer, path)
}

func hashData(original []byte) string {
	sum := sha512.Sum512(original)
	return hex.EncodeToString(sum[:])
}

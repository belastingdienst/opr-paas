package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

func pathToFileList(paths []string) ([]string, error) {
	files := make(map[string]bool)
	for _, path := range paths {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error while walking the path: %e", err)
			} else if info.Mode().IsRegular() {
				files[path] = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	var fileList []string
	for key := range files {
		fileList = append(fileList, key)
	}
	return fileList, nil
}

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

func hashData(original []byte) string {
	sum := sha512.Sum512(original)
	return hex.EncodeToString(sum[:])
}

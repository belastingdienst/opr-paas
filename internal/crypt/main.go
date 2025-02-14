/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package crypt

import (
	"fmt"
	"io"
	"os"
)

func encrypt(publicKey string, paasName string, data []byte) error {
	var encrypted string
	if c, err := NewCryptFromFiles([]string{}, publicKey, paasName); err != nil {
		return err
	} else if encrypted, err = c.Encrypt(data); err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}
	fmt.Println(encrypted)

	return nil
}

func DecryptFromStdin(privateKeys []string, paasName string) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	c, err := NewCryptFromFiles(privateKeys, "", paasName)
	if err != nil {
		return err
	}
	encrypted, err := c.Decrypt(string(data))
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}
	fmt.Println(string(encrypted))
	return nil
}

func EncryptFromStdin(publicKey string, paasName string) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	return encrypt(publicKey, paasName, data)
}

func EncryptFile(publicKey string, paasName string, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return encrypt(publicKey, paasName, data)
}

func GenerateKeyPair(privateKey string, publicKey string) error {
	if privateKey == "" {
		f, err := os.CreateTemp("", "paas")
		if err != nil {
			return fmt.Errorf("privateKey not specified and failed to create temp file: %w", err)
		}

		privateKey = f.Name()
	}

	if publicKey == "" {
		f, err := os.CreateTemp("", "paas")
		if err != nil {
			return fmt.Errorf("privateKey not specified and failed to create temp file: %w", err)
		}

		publicKey = f.Name()
	}

	if _, err := NewGeneratedCrypt(privateKey, publicKey, ""); err != nil {
		return fmt.Errorf("failed to generate new key pair: %w", err)
	}
	return nil
}

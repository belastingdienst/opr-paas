/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package crypt

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
)

func hashedKey(key []byte) []byte {
	hasher := sha1.New()
	hasher.Write(key)
	return hasher.Sum(nil)[:16]
}

func encrypt(publicKey string, paasName string, data []byte) error {
	if c, err := NewCrypt([]string{}, publicKey, paasName); err != nil {
		return err
	} else if encrypted, err := c.Encrypt(data); err != nil {
		return fmt.Errorf("failed to encrypt: %e", err)
	} else {
		fmt.Println(encrypted)
	}
	return nil
}

func DecryptFromStdin(privateKeys []string, paasName string) error {
	if data, err := io.ReadAll(os.Stdin); err != nil {
		return err
	} else {
		if c, err := NewCrypt(privateKeys, "", paasName); err != nil {
			return err

		} else if encrypted, err := c.Decrypt(string(data)); err != nil {
			return fmt.Errorf("failed to decrypt: %e", err)
		} else {
			fmt.Println(string(encrypted))
			return nil
		}
	}
}

func EncryptFromStdin(publicKey string, paasName string) error {
	if data, err := io.ReadAll(os.Stdin); err != nil {
		return err
	} else {
		return encrypt(publicKey, paasName, data)
	}
}

func EncryptFile(publicKey string, paasName string, path string) error {
	if data, err := os.ReadFile(path); err != nil {
		return err
	} else {
		return encrypt(publicKey, paasName, data)
	}
}

func GenerateKeyPair(privateKey string, publicKey string) error {
	if privateKey == "" {
		f, err := os.CreateTemp("", "paas")

		if err != nil {
			return fmt.Errorf("privateKey not specified and failed to create temp file: %e", err)
		}

		privateKey = f.Name()
	}

	if publicKey == "" {
		f, err := os.CreateTemp("", "paas")

		if err != nil {
			return fmt.Errorf("privateKey not specified and failed to create temp file: %e", err)
		}

		publicKey = f.Name()
	}

	if _, err := NewGeneratedCrypt(privateKey, publicKey); err != nil {
		return fmt.Errorf("failed to generate new key pair: %e", err)
	}
	return nil
}

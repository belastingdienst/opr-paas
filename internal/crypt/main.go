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
	c := NewCrypt("", publicKey, paasName)
	if encrypted, err := c.Encrypt(data); err != nil {
		return fmt.Errorf("failed to encrypt: %e", err)
	} else {
		fmt.Println(encrypted)
	}
	return nil
}

func DecryptFromStdin(privateKey string, paasName string) error {
	if data, err := io.ReadAll(os.Stdin); err != nil {
		return err
	} else {
		c := NewCrypt(privateKey, "", paasName)
		if encrypted, err := c.Decrypt(string(data)); err != nil {
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
	if privateKey != "" {

	} else if f, err := os.CreateTemp("", "paas"); err != nil {
		return fmt.Errorf("privateKey not specified and failed to create temp file: %e", err)
	} else {
		privateKey = f.Name()
	}

	if publicKey != "" {

	} else if f, err := os.CreateTemp("", "paas"); err != nil {
		return fmt.Errorf("privateKey not specified and failed to create temp file: %e", err)
	} else {
		publicKey = f.Name()
	}
	if _, err := NewCrypt(privateKey, publicKey, "").GenerateCrypt(); err != nil {
		return fmt.Errorf("failed to generate new key pair: %e", err)
	}
	return nil
}

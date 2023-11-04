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

func encrypt(publicKey string, paasName string, data []byte) {
	c := NewCrypt("", publicKey, paasName)
	if encrypted, err := c.Encrypt(data); err != nil {
		panic(fmt.Sprintf("Failed to encrypt: %e", err))
	} else {
		fmt.Println(encrypted)
	}
}

func DecryptFromStdin(privateKey string, paasName string) {
	if data, err := io.ReadAll(os.Stdin); err != nil {
		panic(err)
	} else {
		c := NewCrypt(privateKey, "", paasName)
		if encrypted, err := c.Decrypt(string(data)); err != nil {
			panic(fmt.Sprintf("Failed to decrypt: %e", err))
		} else {
			fmt.Println(string(encrypted))
		}
	}
}

func EncryptFromStdin(publicKey string, paasName string) {
	if data, err := io.ReadAll(os.Stdin); err != nil {
		panic(err)
	} else {
		encrypt(publicKey, paasName, data)
	}
}

func EncryptFile(publicKey string, paasName string, path string) {
	if data, err := os.ReadFile(path); err != nil {
		panic(err)
	} else {
		encrypt(publicKey, paasName, data)
	}
}

func newFile(envVar string) string {
	if path := os.Getenv(envVar); path != "" {
		return path
	}
	if f, err := os.CreateTemp("", "paas"); err != nil {
		panic(fmt.Sprintf("%s not specified and failed to create temp file: %e", envVar, err))
	} else {
		return f.Name()
	}
}

func GenerateKeyPair() {
	priv := newFile("PAAS_PRIVATE_KEY_PATH")
	pub := newFile("PAAS_PUBLIC_KEY_PATH")

	if _, err := NewCrypt(priv, pub, "").GenerateCrypt(); err != nil {
		panic(fmt.Sprintf("Failed to generate new key pair: %e", err))
	}
}

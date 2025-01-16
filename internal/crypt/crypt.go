/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package crypt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"regexp"

	"github.com/belastingdienst/opr-paas/internal/utils"
)

type Crypt struct {
	privateKeys       CryptPrivateKeys
	publicKeyPath     string
	publicKey         *rsa.PublicKey
	encryptionContext []byte
}

// NewCryptFromFiles returns a Crypt based on the provided privateKeyPaths and publicKeyPath using the encryptionContext
func NewCryptFromFiles(privateKeyPaths []string, publicKeyPath string, encryptionContext string) (*Crypt, error) {
	privateKeys, err := NewPrivateKeysFromFiles(privateKeyPaths)
	if err != nil {
		return nil, err
	}
	return NewCryptFromKeys(privateKeys, publicKeyPath, encryptionContext)
}

// NewCryptFromKeys returns a Crypt based on the provided privateKeys and publicKey using the encryptionContext
func NewCryptFromKeys(privateKeys CryptPrivateKeys, publicKeyPath string, encryptionContext string) (*Crypt, error) {
	if publicKeyPath != "" {
		publicKeyPaths := []string{publicKeyPath}
		if _, err := utils.PathToFileList(publicKeyPaths); err != nil {
			return nil, fmt.Errorf("could not find files in '%v': %w", publicKeyPaths, err)
		}
	}

	return &Crypt{
		privateKeys:       privateKeys,
		publicKeyPath:     publicKeyPath,
		encryptionContext: []byte(encryptionContext),
	}, nil
}

func NewGeneratedCrypt(privateKeyPath string, publicKeyPath string) (*Crypt, error) {
	var c Crypt
	if privateKey, err := rsa.GenerateKey(rand.Reader, 4096); err != nil {
		return nil, fmt.Errorf("unable to generate private key: %w", err)
	} else {
		pk := CryptPrivateKey{
			privateKey:     privateKey,
			privateKeyPath: privateKeyPath,
		}
		c.privateKeys = CryptPrivateKeys{&pk}
		if err := pk.writePrivateKey(); err != nil {
			return nil, err
		}

		c.publicKeyPath = publicKeyPath
		c.publicKey = &privateKey.PublicKey
		if err := c.writePublicKey(); err != nil {
			return nil, err
		}

		return &c, nil
	}
}

func (c *Crypt) writePublicKey() error {
	if c.publicKeyPath == "" {
		return fmt.Errorf("cannot write public key without a specified path")
	}
	if publicKeyBytes, err := x509.MarshalPKIXPublicKey(c.publicKey); err != nil {
		return fmt.Errorf("unable to marshal public key: %w", err)
	} else {
		publicKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: publicKeyBytes,
		})

		if err = os.WriteFile(c.publicKeyPath, publicKeyPEM, 0o600); err != nil {
			return fmt.Errorf("unable to write public key: %w", err)
		}
	}
	fmt.Printf("Public key written to %s\n", c.publicKeyPath)
	return nil
}

func (c *Crypt) getPublicKey() (*rsa.PublicKey, error) {
	if c.publicKey != nil {
		return c.publicKey, nil
	}
	if c.publicKeyPath == "" {
		return nil, fmt.Errorf("cannot get public key without a specified path")
	}
	if publicKeyPEM, err := os.ReadFile(c.publicKeyPath); err != nil {
		panic(err)
	} else if publicKeyBlock, _ := pem.Decode(publicKeyPEM); publicKeyBlock == nil {
		return nil, fmt.Errorf("cannot decode public key")
	} else if publicKey, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes); err != nil {
		return nil, fmt.Errorf("public key invalid: %w", err)
	} else if publicRsaKey, ok := publicKey.(*rsa.PublicKey); !ok {
		return nil, fmt.Errorf("public key not rsa public key")
	} else {
		c.publicKey = publicRsaKey
	}
	return c.publicKey, nil
}

func (c *Crypt) EncryptRsa(secret []byte) (encryptedBytes []byte, err error) {
	if publicKey, err := c.getPublicKey(); err != nil {
		return nil, err
	} else {
		random := rand.Reader
		hash := sha512.New()
		msgLen := len(secret)
		step := publicKey.Size() - 2*hash.Size() - 2
		for start := 0; start < msgLen; start += step {
			finish := start + step
			if finish > msgLen {
				finish = msgLen
			}

			encryptedBlockBytes, err := rsa.EncryptOAEP(hash, random, publicKey, secret[start:finish], c.encryptionContext)
			if err != nil {
				return nil, err
			}

			encryptedBytes = append(encryptedBytes, encryptedBlockBytes...)
		}
		return encryptedBytes, nil
	}
}

func (c *Crypt) Encrypt(secret []byte) (encrypted string, err error) {
	if asymEncrypted, err := c.EncryptRsa(secret); err != nil {
		return "", err
	} else {
		return base64.StdEncoding.EncodeToString(asymEncrypted), nil
	}
}

func (c *Crypt) DecryptRsa(data []byte) (decryptedBytes []byte, err error) {
	if len(c.privateKeys) < 1 {
		return nil, fmt.Errorf("cannot decrypt without any private key")
	}
	for _, pk := range c.privateKeys {
		if decryptedBytes, err = pk.DecryptRsa(data, c.encryptionContext); err != nil {
			continue
		} else {
			return decryptedBytes, nil
		}
	}
	return nil, fmt.Errorf("unable to decrypt data with any of the private keys")
}

func (c Crypt) Decrypt(b64 string) ([]byte, error) {
	// Removing all characters that do not comply to base64 encoding (mainly \n and ' ')
	re := regexp.MustCompile("[^ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=]")
	b64 = re.ReplaceAllLiteralString(b64, "")
	if asymEncrypted, err := base64.StdEncoding.DecodeString(b64); err != nil {
		return nil, err
	} else if decrypted, err := c.DecryptRsa(asymEncrypted); err != nil {
		return nil, err
	} else {
		return decrypted, nil
	}
}

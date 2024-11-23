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
	privateKeys       cryptPrivateKeys
	publicKeyPath     string
	publicKey         *rsa.PublicKey
	encryptionContext []byte
}

type cryptPrivateKey struct {
	privateKeyPath string
	privateKeyPem  []byte
	privateKey     *rsa.PrivateKey
}

type cryptPrivateKeys []cryptPrivateKey

// NewPrivateKeyFromFile returns a cryptPrivateKey from a privateKeyFilePath
func NewPrivateKeyFromFile(privateKeyPath string) (*cryptPrivateKey, error) {
	if privateKeyPath == "" {
		return nil, fmt.Errorf("cannot get private key without a specified path")
	}
	if privateKeyPem, err := os.ReadFile(privateKeyPath); err != nil {
		panic(err)
	} else if privateKeyBlock, _ := pem.Decode(privateKeyPem); privateKeyBlock == nil {
		return nil, fmt.Errorf("cannot decode private key")
	} else if privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes); err != nil {
		return nil, fmt.Errorf("private key invalid: %w", err)
	} else {
		return &cryptPrivateKey{
			privateKeyPath,
			privateKeyPem,
			privateKey,
		}, nil
	}
}

// NewPrivateKeyFromRsa returns a cryptPrivateKey from a rsa.PrivateKey
func NewPrivateKeyFromRsa(privateKey rsa.PrivateKey) *cryptPrivateKey {
	return &cryptPrivateKey{
		"",
		nil,
		&privateKey,
	}
}

func (pk *cryptPrivateKey) writePrivateKey() error {
	if pk.privateKeyPath == "" {
		return fmt.Errorf("cannot write private key without a specified path")
	}
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(pk.privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	if err := os.WriteFile(pk.privateKeyPath, privateKeyPEM, 0o600); err != nil {
		return fmt.Errorf("unable to write private key: %w", err)
	}
	fmt.Printf("Private key written to %s\n", pk.privateKeyPath)
	return nil
}

// getPrivateKey returns the rsa.PrivateKey from the provided cryptPrivateKey. If it is not set yet, it will
// try to load it from the specified filePath. It also checks whether it is a valid PrivateKey.
func (pk cryptPrivateKey) getPrivateKey() (*rsa.PrivateKey, error) {
	// if privateKey is already loaded, return it from the cryptPrivateKey
	if pk.privateKey != nil {
		return pk.privateKey, nil
	}
	// if it is not loaded on the pk, we must load it, in this case from a specific path which hopefully is set
	if pk.privateKeyPath == "" {
		return nil, fmt.Errorf("cannot get private key without a specified path")
	}
	// if set, read file from set path and try to decode
	if privateKeyPEM, err := os.ReadFile(pk.privateKeyPath); err != nil {
		panic(err)
	} else if privateKeyBlock, _ := pem.Decode(privateKeyPEM); privateKeyBlock == nil {
		return nil, fmt.Errorf("cannot decode private key")
		// sanity check if the privatekey is a valid one
	} else if privateRsaKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes); err != nil {
		return nil, fmt.Errorf("private key invalid: %w", err)
	} else {
		pk.privateKey = privateRsaKey
	}
	return pk.privateKey, nil
}

// NewCryptFromFiles returns a Crypt based on the provided privateKeyPaths and publicKeyPath using the encryptionContext
func NewCryptFromFiles(privateKeyPaths []string, publicKeyPath string, encryptionContext string) (*Crypt, error) {
	var privateKeys cryptPrivateKeys

	if files, err := utils.PathToFileList(privateKeyPaths); err != nil {
		return nil, fmt.Errorf("could not find files in '%v': %w", privateKeyPaths, err)
	} else {
		for _, file := range files {
			if pk, err := NewPrivateKeyFromFile(file); err != nil {
				return nil, fmt.Errorf("invalid private key file %s", file)
			} else {
				privateKeys = append(privateKeys, *pk)
			}
		}
	}

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

// NewCryptFromKeys returns a Crypt based on the provided privateKeys and publicKey using the encryptionContext
func NewCryptFromKeys(privateKeys []rsa.PrivateKey, publicKeyPath string, encryptionContext string) (*Crypt, error) {
	var cryptPrivateKeys cryptPrivateKeys

	for _, key := range privateKeys {
		cryptPrivateKeys = append(cryptPrivateKeys, *NewPrivateKeyFromRsa(key))
	}

	if publicKeyPath != "" {
		publicKeyPaths := []string{publicKeyPath}
		if _, err := utils.PathToFileList(publicKeyPaths); err != nil {
			return nil, fmt.Errorf("could not find files in '%v': %w", publicKeyPaths, err)
		}
	}

	return &Crypt{
		privateKeys:       cryptPrivateKeys,
		publicKeyPath:     publicKeyPath,
		encryptionContext: []byte(encryptionContext),
	}, nil
}

func NewGeneratedCrypt(privateKeyPath string, publicKeyPath string) (*Crypt, error) {
	var c Crypt
	if privateKey, err := rsa.GenerateKey(rand.Reader, 4096); err != nil {
		return nil, fmt.Errorf("unable to generate private key: %w", err)
	} else {
		pk := cryptPrivateKey{
			privateKey:     privateKey,
			privateKeyPath: privateKeyPath,
		}
		c.privateKeys = cryptPrivateKeys{pk}
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

func (pk *cryptPrivateKey) DecryptRsa(data []byte, encryptionContext []byte) (decryptedBytes []byte, err error) {
	if privateKey, err := pk.getPrivateKey(); err != nil {
		return nil, err
	} else {
		hash := sha512.New()
		msgLen := len(data)
		step := privateKey.Size()
		random := rand.Reader

		for start := 0; start < msgLen; start += step {
			finish := start + step
			if finish > msgLen {
				finish = msgLen
			}

			decryptedBlockBytes, err := rsa.DecryptOAEP(hash, random, privateKey, data[start:finish], encryptionContext)
			if err != nil {
				return nil, err
			}
			decryptedBytes = append(decryptedBytes, decryptedBlockBytes...)
		}
		return decryptedBytes, nil
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

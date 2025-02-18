package crypt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/belastingdienst/opr-paas/internal/utils"
)

// We can use multiple private keys (for rotation) and store them in a list of PrivateKey's
type CryptPrivateKeys []*CryptPrivateKey

// NewPrivateKeysFromFiles returns a Crypt based on the provided privateKeyPaths
func NewPrivateKeysFromFiles(privateKeyPaths []string) (CryptPrivateKeys, error) {
	var privateKeys CryptPrivateKeys

	files, err := utils.PathToFileList(privateKeyPaths)
	if err != nil {
		return nil, fmt.Errorf("could not find files in '%v': %w", privateKeyPaths, err)
	}
	for _, file := range files {
		pk, err := NewPrivateKeyFromFile(file)
		if err != nil {
			return nil, fmt.Errorf("invalid private key file %s", file)
		}
		privateKeys = append(privateKeys, pk)
	}
	return privateKeys, nil
}

// NewPrivateKeysFromSecretData returns a Crypt based on the provided privateKeyPaths
func NewPrivateKeysFromSecretData(privateKeyData map[string][]byte) (CryptPrivateKeys, error) {
	var privateKeys CryptPrivateKeys

	for name, value := range privateKeyData {
		privateKey, err := NewPrivateKeyFromPem(name, value)
		if err != nil {
			return nil, err
		}
		privateKeys = append(privateKeys, privateKey)
	}
	return privateKeys, nil
}

// Compare checks 2 sets of private keys
func (pks CryptPrivateKeys) Compare(other CryptPrivateKeys) (same bool) {
	if len(pks) != len(other) {
		return false
	}
	for index, key := range pks {
		if !key.privateKey.Equal(other[index]) {
			return false
		}
	}
	return true
}

func (pks CryptPrivateKeys) AsSecretData() (data map[string][]byte) {
	data = map[string][]byte{}
	for _, key := range pks {
		data[key.privateKeyPath] = key.privateKeyPem
	}
	return data
}

// A CryptPrivateKey is used for decryption of encrypted secrets
type CryptPrivateKey struct {
	privateKeyPath string
	privateKeyPem  []byte
	privateKey     *rsa.PrivateKey
}

// NewPrivateKeyFromFile returns a CryptPrivateKey from a privateKeyFilePath
func NewPrivateKeyFromFile(privateKeyPath string) (*CryptPrivateKey, error) {
	if privateKeyPath == "" {
		return nil, errors.New("cannot get private key without a specified path")
	}
	if privateKeyPem, err := os.ReadFile(privateKeyPath); err != nil {
		panic(err)
	} else {
		return NewPrivateKeyFromPem(privateKeyPath, privateKeyPem)
	}
}

// NewPrivateKeyFromPem returns a CryptPrivateKey from a privateKeyFilePath
func NewPrivateKeyFromPem(privateKeyPath string, privateKeyPem []byte) (*CryptPrivateKey, error) {
	var privateKey *rsa.PrivateKey
	return &CryptPrivateKey{
		privateKeyPath,
		privateKeyPem,
		privateKey,
	}, nil
}

func (pk *CryptPrivateKey) writePrivateKey() error {
	if pk.privateKeyPath == "" {
		return errors.New("cannot write private key without a specified path")
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

// getPrivateKey returns the rsa.PrivateKey from the provided CryptPrivateKey. If it is not set yet, it will
// try to load it from the specified filePath. It also checks whether it is a valid PrivateKey.
func (pk *CryptPrivateKey) getPrivateKey() (privateKey *rsa.PrivateKey, err error) {
	// if privateKey is already loaded, return it from the CryptPrivateKey
	if pk.privateKey != nil {
		return pk.privateKey, nil
	} else if len(pk.privateKeyPem) == 0 {
		return nil, errors.New("invalid private key (Pem not set)")
	}

	// load privateKey from privateKeyPem
	privateKeyBlock, _ := pem.Decode(pk.privateKeyPem)
	if privateKeyBlock == nil {
		return nil, errors.New("cannot decode private key")
		// sanity check if the privatekey is a valid one
	}
	privateRsaKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("private key invalid: %w", err)
	}
	pk.privateKey = privateRsaKey
	return pk.privateKey, nil
}

func (pk *CryptPrivateKey) DecryptRsa(data []byte, encryptionContext []byte) (decryptedBytes []byte, err error) {
	privateKey, err := pk.getPrivateKey()
	if err != nil {
		return nil, err
	}
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

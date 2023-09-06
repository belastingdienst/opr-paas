package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"
)

type Crypt struct {
	privateKeyPath string
	privateKey     *rsa.PrivateKey
	publicKeyPath  string
	publicKey      *rsa.PublicKey
	aesKey         []byte
}

func NewCrypt(privateKeyPath string, publicKeyPath string, symmetricKey string) *Crypt {
	return &Crypt{
		privateKeyPath: privateKeyPath,
		publicKeyPath:  publicKeyPath,
		aesKey:         []byte(symmetricKey),
	}
}

func (c Crypt) Generate() (*Crypt, error) {
	if privateKey, err := rsa.GenerateKey(rand.Reader, 4096); err != nil {
		return nil, fmt.Errorf("unable to generate private key: %e", err)
	} else {
		c.privateKey = privateKey
		c.publicKey = &privateKey.PublicKey
	}
	if err := c.writePrivateKey(); err != nil {
		return nil, err
	}
	if err := c.writePublicKey(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Crypt) writePrivateKey() error {
	if c.privateKeyPath == "" {
		return fmt.Errorf("cannot write private key without a specified path")
	}
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(c.privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	if err := os.WriteFile(c.privateKeyPath, privateKeyPEM, 0644); err != nil {
		return fmt.Errorf("unable to write private key: %e", err)
	}
	fmt.Printf("Private key written to %s\n", c.privateKeyPath)
	return nil
}

func (c *Crypt) writePublicKey() error {
	if c.publicKeyPath == "" {
		return fmt.Errorf("cannot write public key without a specified path")
	}
	if publicKeyBytes, err := x509.MarshalPKIXPublicKey(c.publicKey); err != nil {
		return fmt.Errorf("unable to marshal public key: %e", err)
	} else {
		publicKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: publicKeyBytes,
		})
		if err = os.WriteFile(c.publicKeyPath, publicKeyPEM, 0644); err != nil {
			return fmt.Errorf("unable to write public key: %e", err)
		}
	}
	fmt.Printf("Public key written to %s\n", c.publicKeyPath)
	return nil
}

func (c Crypt) EncryptAes(decrypted []byte) ([]byte, error) {
	if ci, err := aes.NewCipher(hashedKey(c.aesKey)); err != nil {
		return nil, fmt.Errorf("could not create new AES cypher: %e", err)
	} else if gcm, err := cipher.NewGCM(ci); err != nil {
		// gcm or Galois/Counter Mode, is a mode of operation
		// for symmetric key cryptographic block ciphers
		// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
		return nil, fmt.Errorf("could not create new AES GCM: %e", err)
	} else {
		// creates a new byte array the size of the nonce
		// which must be passed to Seal
		nonce := make([]byte, gcm.NonceSize())

		// populates our nonce with a cryptographically secure
		// random sequence
		if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
			fmt.Println(err)
		}

		// here we encrypt our text using the Seal function
		// Seal encrypts and authenticates plaintext, authenticates the
		// additional data and appends the result to dst, returning the updated
		// slice. The nonce must be NonceSize() bytes long and unique for all
		// time, for a given key.
		return gcm.Seal(nonce, nonce, decrypted, nil), nil
	}
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
		return nil, fmt.Errorf("public key invalid: %e", err)
	} else if publicRsaKey, ok := publicKey.(*rsa.PublicKey); !ok {
		return nil, fmt.Errorf("public key not rsa public key")
	} else {
		c.publicKey = publicRsaKey
	}
	return c.publicKey, nil
}

func (c *Crypt) EncryptRsa(secret []byte) (encrypted []byte, err error) {
	plaintext := secret
	if publicKey, err := c.getPublicKey(); err != nil {
		return nil, err
	} else if encrypted, err = rsa.EncryptPKCS1v15(rand.Reader, publicKey, plaintext); err != nil {
		return nil, fmt.Errorf("unable to encrypt secret data with rsa: %e", err)
	} else {
		return encrypted, nil
	}
}

func (c *Crypt) Encrypt(secret []byte) (encrypted string, err error) {
	if symEncrypted, err := c.EncryptAes(secret); err != nil {
		return "", err
	} else if asymEncrypted, err := c.EncryptRsa(symEncrypted); err != nil {
		return "", err
	} else {
		return base64.StdEncoding.EncodeToString(asymEncrypted), nil
	}
}

func (c *Crypt) getPrivateKey() (*rsa.PrivateKey, error) {
	if c.privateKey != nil {
		return c.privateKey, nil
	}
	if c.privateKeyPath == "" {
		return nil, fmt.Errorf("cannot get private key without a specified path")
	}
	if privateKeyPEM, err := os.ReadFile(c.privateKeyPath); err != nil {
		panic(err)
	} else if privateKeyBlock, _ := pem.Decode(privateKeyPEM); privateKeyBlock == nil {
		return nil, fmt.Errorf("cannot decode private key")
	} else if privateRsaKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes); err != nil {
		return nil, fmt.Errorf("private key invalid: %e", err)
	} else {
		c.privateKey = privateRsaKey
	}
	return c.privateKey, nil
}

func (c *Crypt) DecryptRsa(data []byte) ([]byte, error) {
	if privateKey, err := c.getPrivateKey(); err != nil {
		return nil, err
	} else if plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, data); err != nil {
		return nil, fmt.Errorf("unable to decrypt secret data with rsa: %e", err)
	} else {
		return plaintext, nil
	}
}

func (c Crypt) DecryptAes(encrypted []byte) ([]byte, error) {

	if ci, err := aes.NewCipher(hashedKey(c.aesKey)); err != nil {
		return nil, fmt.Errorf("could not create new AES cypher: %e", err)
	} else if gcm, err := cipher.NewGCM(ci); err != nil {
		return nil, fmt.Errorf("could not create new AES GCM: %e", err)
	} else if nonceSize := gcm.NonceSize(); len(encrypted) < nonceSize {
		return nil, fmt.Errorf("AES error: invalid encrypted data (%d smaller than nonce %d)", len(encrypted), nonceSize)
	} else {
		nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
		if plaintext, err := gcm.Open(nil, nonce, ciphertext, nil); err != nil {
			return nil, fmt.Errorf("AES error: decryption failed: %e", err)
		} else {
			return plaintext, nil
		}
	}
}

func (c Crypt) Decrypt(b64 string) ([]byte, error) {
	if asymEncrypted, err := base64.StdEncoding.DecodeString(b64); err != nil {
		return nil, err
	} else if symEncrypted, err := c.DecryptRsa(asymEncrypted); err != nil {
		return nil, err
	} else if decrypted, err := c.DecryptAes(symEncrypted); err != nil {
		return nil, err
	} else {
		return decrypted, nil
	}
}

package gbrain

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

type credentialCipher struct {
	aead cipher.AEAD
}

func newCredentialCipher(encodedKey string) (*credentialCipher, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(key) != 32 {
		return nil, errors.New("BRAINHUB_WRITER_CREDENTIAL_KEY must be base64-encoded 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &credentialCipher{aead: aead}, nil
}

func (c *credentialCipher) encrypt(brainID, clientID, secret string) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, []byte(secret), credentialAAD(brainID, clientID)), nil
}

func (c *credentialCipher) decrypt(brainID, clientID string, ciphertext []byte) (string, error) {
	if len(ciphertext) < c.aead.NonceSize()+c.aead.Overhead() {
		return "", errors.New("writer credential ciphertext is invalid")
	}
	nonce := ciphertext[:c.aead.NonceSize()]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext[c.aead.NonceSize():], credentialAAD(brainID, clientID))
	if err != nil {
		return "", fmt.Errorf("decrypt writer credential: %w", err)
	}
	return string(plaintext), nil
}

func credentialAAD(brainID, clientID string) []byte {
	return []byte(brainID + "\x00" + clientID)
}

// There is intentionally no key version in the v1 ciphertext or table: key
// rotation is not implemented. A future migration can add key_version without
// changing these rows; recovery today revokes and reissues each writer client.

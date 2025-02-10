package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

var (
	ErrInvalidKeySize = errors.New("invalid key size")
	ErrEncryption     = errors.New("encryption failed")
	ErrDecryption     = errors.New("decryption failed")
)

// Encryptor provides a generic interface for encryption/decryption
type Encryptor interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}

// NewAESEncryptor creates a new AES-GCM encryptor
func NewAESEncryptor(key []byte) (Encryptor, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrInvalidKeySize
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrEncryption
	}

	return &aesEncryptor{
		gcm: gcm,
	}, nil
}

type aesEncryptor struct {
	gcm cipher.AEAD
}

func (a *aesEncryptor) Encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, a.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, ErrEncryption
	}

	return a.gcm.Seal(nonce, nonce, data, nil), nil
}

func (a *aesEncryptor) Decrypt(data []byte) ([]byte, error) {
	nonceSize := a.gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrDecryption
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := a.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryption
	}

	return plaintext, nil
}

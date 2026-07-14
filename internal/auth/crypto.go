package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// DecodeEncKey decodes CONFIG_ENC_KEY (hex-encoded, 64 chars = 32 bytes) for
// AES-256-GCM. Returns an error if the key is missing or the wrong length —
// callers should treat this as "AI Provider Config unavailable" (EP-18
// ST-18.4), not crash the whole app.
func DecodeEncKey(hexKey string) ([]byte, error) {
	if hexKey == "" {
		return nil, errors.New("crypto: CONFIG_ENC_KEY kosong")
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("crypto: CONFIG_ENC_KEY bukan hex valid: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: CONFIG_ENC_KEY harus 32 byte (64 karakter hex), dapat %d byte", len(key))
	}
	return key, nil
}

// Encrypt encrypts plaintext with AES-256-GCM. A fresh random nonce is
// generated per call and prepended to the ciphertext, then the whole thing
// is base64-encoded so it fits in a single TEXT column
// (ai_provider_setting.api_key_encrypted).
func Encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt reverses Encrypt.
func Decrypt(encoded string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: decode base64: %w", err)
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("crypto: ciphertext terlalu pendek")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt gagal: %w", err)
	}
	return string(plaintext), nil
}

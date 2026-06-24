package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// Hash returns a bcrypt hash of plain.
func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth.Hash: %w", err)
	}
	return string(b), nil
}

// Verify returns nil if plain matches hash, otherwise an error.
func Verify(hash, plain string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return fmt.Errorf("auth.Verify: %w", err)
	}
	return nil
}

// GenerateTempPassword returns a cryptographically-random URL-safe password (~16 chars).
func GenerateTempPassword() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth.GenerateTempPassword: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

package auth

import (
	"strings"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key, err := DecodeEncKey("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	if err != nil {
		t.Fatalf("DecodeEncKey: %v", err)
	}
	return key
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := testKey(t)
	plaintext := "sk-test-api-key-abcdef123456"

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ciphertext == plaintext {
		t.Fatal("ciphertext must not equal plaintext")
	}
	if strings.Contains(ciphertext, plaintext) {
		t.Fatal("ciphertext must not leak plaintext substring")
	}

	got, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("Decrypt(Encrypt(x)) = %q, want %q", got, plaintext)
	}
}

func TestEncrypt_NonceDiffersEachCall(t *testing.T) {
	key := testKey(t)
	plaintext := "sk-same-key"

	c1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt 1: %v", err)
	}
	c2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt 2: %v", err)
	}
	if c1 == c2 {
		t.Fatal("two encryptions of the same plaintext must differ (random nonce)")
	}
}

func TestDecrypt_CorruptedData(t *testing.T) {
	key := testKey(t)
	ciphertext, err := Encrypt("sk-test", key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	corrupted := ciphertext[:len(ciphertext)-4] + "abcd"

	if _, err := Decrypt(corrupted, key); err == nil {
		t.Fatal("Decrypt should fail on corrupted ciphertext, not silently return garbage")
	}
}

func TestDecodeEncKey_Empty(t *testing.T) {
	if _, err := DecodeEncKey(""); err == nil {
		t.Fatal("DecodeEncKey should error on empty key")
	}
}

func TestDecodeEncKey_WrongLength(t *testing.T) {
	if _, err := DecodeEncKey("abcd"); err == nil {
		t.Fatal("DecodeEncKey should error on a key shorter than 32 bytes")
	}
}

func TestDecodeEncKey_NotHex(t *testing.T) {
	if _, err := DecodeEncKey("not-valid-hex-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"); err == nil {
		t.Fatal("DecodeEncKey should error on invalid hex")
	}
}

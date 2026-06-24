package auth

import "testing"

func TestHashAndVerify(t *testing.T) {
	plain := "s3cr3tPassw0rd!"

	h, err := Hash(plain)
	if err != nil {
		t.Fatalf("Hash error: %v", err)
	}
	if h == "" {
		t.Fatal("Hash returned empty string")
	}

	if err := Verify(h, plain); err != nil {
		t.Errorf("Verify correct password failed: %v", err)
	}
}

func TestVerify_WrongPassword(t *testing.T) {
	h, _ := Hash("correct")
	if err := Verify(h, "wrong"); err == nil {
		t.Error("expected error for wrong password, got nil")
	}
}

func TestHash_ProducesDifferentSalts(t *testing.T) {
	h1, _ := Hash("same")
	h2, _ := Hash("same")
	if h1 == h2 {
		t.Error("two hashes of same input should differ (bcrypt salts)")
	}

	// Both must still verify against the original plain.
	if err := Verify(h1, "same"); err != nil {
		t.Errorf("verify h1 failed: %v", err)
	}
	if err := Verify(h2, "same"); err != nil {
		t.Errorf("verify h2 failed: %v", err)
	}
}

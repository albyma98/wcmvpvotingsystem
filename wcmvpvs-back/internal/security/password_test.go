package security

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("my-secret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !IsArgon2idHash(hash) {
		t.Fatalf("expected argon2id hash, got %q", hash)
	}
	ok, err := VerifyPassword("my-secret", hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !ok {
		t.Fatal("expected password verification success")
	}
}

func TestVerifyPasswordWrongSecret(t *testing.T) {
	hash, err := HashPassword("my-secret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	ok, err := VerifyPassword("wrong", hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if ok {
		t.Fatal("expected password verification failure")
	}
}

func TestLegacyHashDetection(t *testing.T) {
	legacy := "2bb80d537b1da3e38bd30361aa855686bde0eacd7162fef6a25fe97bf527a25b"
	if !IsLegacySHA256Hash(legacy) {
		t.Fatal("expected legacy SHA-256 hash detection")
	}
	if !VerifyLegacySHA256Password("secret", "2bb80d537b1da3e38bd30361aa855686bde0eacd7162fef6a25fe97bf527a25b") {
		t.Fatal("expected legacy SHA-256 verification success")
	}
	if VerifyLegacySHA256Password("wrong", legacy) {
		t.Fatal("expected legacy SHA-256 verification failure")
	}
}

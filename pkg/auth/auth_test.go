package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" || hash == "password123" {
		t.Fatal("password must be hashed")
	}
	if !VerifyPassword(hash, "password123") {
		t.Fatal("expected password to match")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatal("expected mismatch for wrong password")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret"
	token, expiresAt, err := GenerateToken(42, "u@example.com", "user", secret)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	if token == "" || expiresAt.IsZero() {
		t.Fatal("expected non-empty token and expiry")
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if claims.UserID != 42 || claims.Email != "u@example.com" || claims.Username != "user" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	if _, err := ValidateToken(token, "other-secret"); err == nil {
		t.Fatal("expected validation error for wrong secret")
	}
}

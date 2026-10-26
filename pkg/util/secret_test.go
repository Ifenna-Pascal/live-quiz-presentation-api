package util

import "testing"

func TestNewSecretMatchesItsHash(t *testing.T) {
	secret, hash, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(secret) < 40 {
		t.Fatalf("secret too short: %d", len(secret))
	}
	if !SecretMatchesHash(secret, hash) {
		t.Fatal("secret should match its own hash")
	}
	if SecretMatchesHash(secret+"x", hash) {
		t.Fatal("a different secret must not match")
	}
}

func TestSecretsAreUnique(t *testing.T) {
	a, _, _ := NewSecret()
	b, _, _ := NewSecret()
	if a == b {
		t.Fatal("two secrets should never be equal")
	}
}

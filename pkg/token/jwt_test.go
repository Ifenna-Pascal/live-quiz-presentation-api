package token

import (
	"testing"
	"time"
)

const testKey = "test-secret-key-that-is-long-enough-123"

func TestCreateAndVerifyToken(t *testing.T) {
	maker, err := NewJWTMaker(testKey)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := maker.CreateToken("user-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := maker.VerifyToken(tok)
	if err != nil || subject != "user-1" {
		t.Fatalf("got %q, %v; want user-1, nil", subject, err)
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	maker, _ := NewJWTMaker(testKey)
	tok, _ := maker.CreateToken("user-1", -time.Second)
	if _, err := maker.VerifyToken(tok); err != ErrInvalidToken {
		t.Fatalf("got %v, want ErrInvalidToken", err)
	}
}

func TestTokenFromAnotherKeyIsRejected(t *testing.T) {
	a, _ := NewJWTMaker(testKey)
	b, _ := NewJWTMaker(testKey + "-other")
	tok, _ := a.CreateToken("user-1", time.Minute)
	if _, err := b.VerifyToken(tok); err != ErrInvalidToken {
		t.Fatalf("got %v, want ErrInvalidToken", err)
	}
}

func TestShortKeyIsRefused(t *testing.T) {
	if _, err := NewJWTMaker("short"); err == nil {
		t.Fatal("expected an error for a short key")
	}
}

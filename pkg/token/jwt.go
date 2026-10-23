package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// minSecretKeySize is the shortest HMAC key we accept. HS256 is only as
// strong as its key, so short keys are rejected at startup.
const minSecretKeySize = 32

// ErrInvalidToken is returned when a token is malformed, badly signed or expired.
var ErrInvalidToken = errors.New("invalid or expired token")

// Maker creates and verifies signed access tokens.
type Maker interface {
	CreateToken(userID string, duration time.Duration) (string, error)
	// VerifyToken checks the signature and expiry of token and returns the
	// user ID stored in it.
	VerifyToken(token string) (string, error)
}

type jwtMaker struct {
	secretKey []byte
}

// NewJWTMaker returns a Maker that signs tokens with HS256 using secretKey.
func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, errors.New("token secret key must be at least 32 characters")
	}
	return &jwtMaker{secretKey: []byte(secretKey)}, nil
}

// CreateToken returns a JWT whose subject is userID and which expires after duration.
func (m *jwtMaker) CreateToken(userID string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secretKey)
}

// VerifyToken parses token, checks that it was signed with HS256 by this
// Maker and has not expired, and returns its subject (the user ID).
func (m *jwtMaker) VerifyToken(token string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return m.secretKey, nil
	},
		// Only accept HS256, so a token cannot pick a different algorithm (e.g. "none").
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		// Reject tokens that have no expiry at all.
		jwt.WithExpirationRequired(),
	)
	if err != nil || claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}

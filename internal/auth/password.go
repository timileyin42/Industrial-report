// Package auth provides password/secret hashing, JWT issuance, and Echo
// middleware for role- and site-scoped access control.
package auth

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// GenerateDeviceSecret returns a random 32-byte secret, base64url-encoded.
// Callers must show this to the caller exactly once — it is never
// retrievable again, only its bcrypt hash is stored.
func GenerateDeviceSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashSecret and VerifySecret back both devices.secret_hash and
// users.password_hash — one hashing primitive, two callers.
func HashSecret(plaintext string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifySecret(hash, plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}

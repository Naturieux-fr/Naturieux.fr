// Package auth provides password hashing and stateless session tokens for
// the admin back-office.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidToken is returned when a token is malformed, tampered with, or
// expired.
var ErrInvalidToken = errors.New("invalid or expired token")

// HashPassword returns the bcrypt hash of a plaintext password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword reports whether password matches the bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// IssueToken returns a stateless token binding a subject (player id) to an
// expiry, signed with the secret. Format: base64(subject|expiryUnix).hexMAC.
func IssueToken(subject, secret string, ttl time.Duration, now time.Time) string {
	payload := subject + "|" + strconv.FormatInt(now.Add(ttl).Unix(), 10)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return encoded + "." + sign(encoded, secret)
}

// VerifyToken validates a token's signature and expiry, returning its
// subject (player id).
func VerifyToken(token, secret string, now time.Time) (string, error) {
	encoded, mac, ok := strings.Cut(token, ".")
	if !ok {
		return "", ErrInvalidToken
	}
	if !hmac.Equal([]byte(mac), []byte(sign(encoded, secret))) {
		return "", ErrInvalidToken
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidToken
	}
	subject, expStr, ok := strings.Cut(string(raw), "|")
	if !ok {
		return "", ErrInvalidToken
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || now.Unix() >= exp {
		return "", ErrInvalidToken
	}
	return subject, nil
}

// sign returns the hex HMAC-SHA256 of msg under secret.
func sign(msg, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

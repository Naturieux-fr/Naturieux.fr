package auth_test

import (
	"testing"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/auth"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := auth.HashPassword("s3cret-pass")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "s3cret-pass" {
		t.Error("password stored in clear")
	}
	if !auth.CheckPassword(hash, "s3cret-pass") {
		t.Error("CheckPassword() rejected the correct password")
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Error("CheckPassword() accepted a wrong password")
	}
}

func TestToken_RoundTrip(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token := auth.IssueToken("player-1", "secret", time.Hour, now)

	subject, err := auth.VerifyToken(token, "secret", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}
	if subject != "player-1" {
		t.Errorf("subject = %s, want player-1", subject)
	}
}

func TestToken_Expired(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token := auth.IssueToken("player-1", "secret", time.Hour, now)

	if _, err := auth.VerifyToken(token, "secret", now.Add(2*time.Hour)); err == nil {
		t.Error("VerifyToken() should reject an expired token")
	}
}

func TestToken_WrongSecret(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token := auth.IssueToken("player-1", "secret", time.Hour, now)

	if _, err := auth.VerifyToken(token, "other-secret", now); err == nil {
		t.Error("VerifyToken() should reject a token signed with another secret")
	}
}

func TestToken_Tampered(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token := auth.IssueToken("player-1", "secret", time.Hour, now)

	if _, err := auth.VerifyToken(token+"x", "secret", now); err == nil {
		t.Error("VerifyToken() should reject a tampered token")
	}
	if _, err := auth.VerifyToken("garbage", "secret", now); err == nil {
		t.Error("VerifyToken() should reject a malformed token")
	}
}

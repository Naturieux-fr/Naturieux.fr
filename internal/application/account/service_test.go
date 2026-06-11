package account_test

import (
	"context"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/account"
)

func newService(t *testing.T, mode account.Mode) *account.Service {
	t.Helper()
	db, err := sqlite.Open(t.TempDir() + "/acc.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := sqlite.NewPlayerRepository(db)
	return account.NewService(repo, repo, "test-secret", mode)
}

func TestAccount_RegisterAndLogin(t *testing.T) {
	s := newService(t, account.Open)
	ctx := context.Background()

	p, token, err := s.Register(ctx, "Alice", "secret123", "")
	if err != nil || token == "" || p == nil {
		t.Fatalf("Register() = (%v, %q, %v)", p, token, err)
	}

	if _, _, err := s.Register(ctx, "Alice", "another1", ""); err != account.ErrUsernameTaken {
		t.Errorf("duplicate Register = %v, want ErrUsernameTaken", err)
	}
	if _, _, err := s.Register(ctx, "Bob", "short", ""); err != account.ErrWeakPassword {
		t.Errorf("weak password = %v, want ErrWeakPassword", err)
	}

	if _, tok, err := s.Login(ctx, "Alice", "secret123"); err != nil || tok == "" {
		t.Errorf("Login(correct) = (%q, %v), want token", tok, err)
	}
	if _, _, err := s.Login(ctx, "Alice", "wrong"); err != account.ErrInvalidCredentials {
		t.Errorf("Login(wrong) = %v, want ErrInvalidCredentials", err)
	}
	if _, _, err := s.Login(ctx, "Ghost", "whatever"); err != account.ErrInvalidCredentials {
		t.Errorf("Login(unknown) = %v, want ErrInvalidCredentials", err)
	}
}

func TestAccount_InviteOnly(t *testing.T) {
	s := newService(t, account.Invite)
	ctx := context.Background()

	if _, _, err := s.Register(ctx, "Carol", "secret123", ""); err != account.ErrInviteRequired {
		t.Errorf("Register without invite = %v, want ErrInviteRequired", err)
	}
	if _, _, err := s.Register(ctx, "Carol", "secret123", "bogus-token"); err != account.ErrInviteRequired {
		t.Errorf("Register with bad invite = %v, want ErrInviteRequired", err)
	}

	invite := s.IssueInvite()
	if _, token, err := s.Register(ctx, "Carol", "secret123", invite); err != nil || token == "" {
		t.Errorf("Register with valid invite = (%q, %v), want success", token, err)
	}
}

package account_test

import (
	"context"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/account"
)

func newSvc(t *testing.T) *account.Service {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := sqlite.NewPlayerRepository(db)
	s := account.NewService(repo, repo, sqlite.NewInviteRepository(db), "secret", account.Open)
	s.SetResetStore(sqlite.NewResetRepository(db))
	return s
}

func TestService_RegisterLoginMeReset(t *testing.T) {
	s := newSvc(t)
	ctx := context.Background()

	player, token, err := s.Register(ctx, "Alice", "secret1", "")
	if err != nil || token == "" {
		t.Fatalf("register: %v", err)
	}
	if id, err := s.Authenticate(token); err != nil || id != player.ID() {
		t.Errorf("authenticate = %q,%v", id, err)
	}
	if _, _, err := s.Login(ctx, "Alice", "secret1"); err != nil {
		t.Errorf("login: %v", err)
	}
	if _, _, err := s.Login(ctx, "Alice", "wrong"); err != account.ErrInvalidCredentials {
		t.Errorf("bad login = %v", err)
	}
	id, name, role, err := s.Me(ctx, token)
	if err != nil || id != player.ID() || name != "Alice" || role != "player" {
		t.Errorf("me = %q,%q,%q,%v", id, name, role, err)
	}

	// Password reset.
	rtok, err := s.IssueReset(ctx, player.ID())
	if err != nil {
		t.Fatalf("issue reset: %v", err)
	}
	if _, _, err := s.ResetPassword(ctx, rtok, "newpass1"); err != nil {
		t.Errorf("reset: %v", err)
	}
	if _, _, err := s.ResetPassword(ctx, rtok, "again12"); err != account.ErrInvalidReset {
		t.Errorf("reused reset = %v", err)
	}
	if _, _, err := s.Login(ctx, "Alice", "newpass1"); err != nil {
		t.Errorf("login new pw: %v", err)
	}
}

func TestService_RegisterValidation(t *testing.T) {
	s := newSvc(t)
	ctx := context.Background()
	if _, _, err := s.Register(ctx, "A", "secret1", ""); err != account.ErrBadUsername {
		t.Errorf("short username = %v", err)
	}
	if _, _, err := s.Register(ctx, "Bob", "123", ""); err != account.ErrWeakPassword {
		t.Errorf("weak pw = %v", err)
	}
	_, _, _ = s.Register(ctx, "Carol", "secret1", "")
	if _, _, err := s.Register(ctx, "Carol", "secret1", ""); err != account.ErrUsernameTaken {
		t.Errorf("dup = %v", err)
	}
}

func TestService_Invites(t *testing.T) {
	s := newSvc(t)
	ctx := context.Background()
	if s.Mode() != account.Open {
		t.Errorf("Mode = %q", s.Mode())
	}
	tok, err := s.IssueInvite(ctx, "admin")
	if err != nil || tok == "" {
		t.Fatalf("IssueInvite: %v", err)
	}
	if list, _ := s.ListInvites(ctx); len(list) != 1 {
		t.Errorf("ListInvites = %d", len(list))
	}
	if err := s.RevokeInvite(ctx, tok); err != nil {
		t.Errorf("RevokeInvite: %v", err)
	}
}

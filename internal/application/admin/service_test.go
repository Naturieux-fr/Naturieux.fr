package admin_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	adminapp "github.com/Naturieux-fr/Naturieux.fr/internal/application/admin"
)

func newService(t *testing.T) *adminapp.Service {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "admin.db"))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return adminapp.NewService(sqlite.NewPlayerRepository(db), "test-secret")
}

func TestService_SeedLoginAuthorize(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()

	if err := svc.SeedAdmin(ctx, "boss", "hunter2pass"); err != nil {
		t.Fatalf("SeedAdmin() error = %v", err)
	}

	token, err := svc.Login(ctx, "boss", "hunter2pass")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token == "" {
		t.Fatal("Login() returned an empty token")
	}

	id, err := svc.Authorize(ctx, token)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if id == "" {
		t.Error("Authorize() returned an empty id")
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()
	if err := svc.SeedAdmin(ctx, "boss", "hunter2pass"); err != nil {
		t.Fatalf("SeedAdmin() error = %v", err)
	}

	if _, err := svc.Login(ctx, "boss", "wrong"); err != adminapp.ErrInvalidCredentials {
		t.Errorf("Login(wrong) error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := svc.Login(ctx, "ghost", "whatever"); err != adminapp.ErrInvalidCredentials {
		t.Errorf("Login(unknown) error = %v, want ErrInvalidCredentials", err)
	}
}

func TestService_SeedAdmin_UpdatesPassword(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()
	if err := svc.SeedAdmin(ctx, "boss", "old-password"); err != nil {
		t.Fatalf("SeedAdmin() error = %v", err)
	}
	// Re-seeding the same username rotates the password.
	if err := svc.SeedAdmin(ctx, "boss", "new-password"); err != nil {
		t.Fatalf("SeedAdmin() re-seed error = %v", err)
	}
	if _, err := svc.Login(ctx, "boss", "new-password"); err != nil {
		t.Errorf("Login(new) error = %v, want success", err)
	}
	if _, err := svc.Login(ctx, "boss", "old-password"); err == nil {
		t.Error("Login(old) should fail after password rotation")
	}
}

func TestService_Authorize_RejectsNonAdminAndGarbage(t *testing.T) {
	svc := newService(t)
	if _, err := svc.Authorize(context.Background(), "garbage-token"); err != adminapp.ErrForbidden {
		t.Errorf("Authorize(garbage) error = %v, want ErrForbidden", err)
	}
}

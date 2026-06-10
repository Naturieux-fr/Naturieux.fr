// Package admin holds the application service for the back-office:
// admin authentication and (via the photo store) photo management.
package admin

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Naturieux-fr/Naturieux.fr/internal/auth"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// ErrInvalidCredentials is returned when a login fails.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrForbidden is returned when an authenticated user lacks the admin role.
var ErrForbidden = errors.New("forbidden")

// roleAdmin is the role granted to back-office users.
const roleAdmin = "admin"

// tokenTTL is how long an admin session token stays valid.
const tokenTTL = 12 * time.Hour

// clock returns the current time; overridable in tests.
type clock func() time.Time

// Service authenticates administrators and issues session tokens.
type Service struct {
	accounts ports.AccountStore
	secret   string
	now      clock
}

// NewService creates an admin auth service signing tokens with secret.
func NewService(accounts ports.AccountStore, secret string) *Service {
	return &Service{accounts: accounts, secret: secret, now: time.Now}
}

// SeedAdmin creates or updates the admin account with the given credentials.
func (s *Service) SeedAdmin(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return errors.New("admin username and password are required")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return s.accounts.UpsertAdmin(ctx, uuid.New().String(), username, hash, s.now().UTC().Format(time.RFC3339Nano))
}

// Login verifies credentials and returns a session token on success.
func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	acc, err := s.accounts.Credentials(ctx, username)
	if err != nil || acc.Role != roleAdmin || !auth.CheckPassword(acc.PasswordHash, password) {
		// Uniform error: never reveal whether the username exists.
		return "", ErrInvalidCredentials
	}
	return auth.IssueToken(acc.ID, s.secret, tokenTTL, s.now()), nil
}

// Authorize validates a session token and confirms the subject is still an
// admin, returning the admin's player id.
func (s *Service) Authorize(ctx context.Context, token string) (string, error) {
	id, err := auth.VerifyToken(token, s.secret, s.now())
	if err != nil {
		return "", ErrForbidden
	}
	role, err := s.accounts.Role(ctx, id)
	if err != nil || role != roleAdmin {
		return "", ErrForbidden
	}
	return id, nil
}

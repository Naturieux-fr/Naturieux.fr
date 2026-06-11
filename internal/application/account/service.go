// Package account provides player registration and login: real accounts with
// passwords, with either open self-service signup or admin-invite-only signup.
package account

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Naturieux-fr/Naturieux.fr/internal/auth"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// Errors returned by the service.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInviteRequired     = errors.New("a valid invitation is required to register")
	ErrWeakPassword       = errors.New("password must be at least 6 characters")
	ErrBadUsername        = errors.New("username must be 2 to 20 characters")
)

// Mode is the registration policy.
type Mode string

const (
	// Open lets anyone create an account from the site.
	Open Mode = "open"
	// Invite requires an admin-issued invitation link.
	Invite Mode = "invite"
)

const (
	sessionTTL = 30 * 24 * time.Hour // players stay logged in for a month
	inviteTTL  = 14 * 24 * time.Hour // an invitation link expires after two weeks
)

// Service registers and authenticates players.
type Service struct {
	accounts ports.AccountStore
	players  ports.PlayerRepository
	invites  ports.InviteStore
	secret   string
	mode     Mode
	now      func() time.Time
}

// NewService creates the account service. mode selects open or invite signup.
func NewService(accounts ports.AccountStore, players ports.PlayerRepository, invites ports.InviteStore, secret string, mode Mode) *Service {
	if mode != Invite {
		mode = Open
	}
	return &Service{accounts: accounts, players: players, invites: invites, secret: secret, mode: mode, now: time.Now}
}

// Mode reports the active registration policy.
func (s *Service) Mode() Mode { return s.mode }

// Register creates a new player account and returns the player with a session
// token. In invite mode a valid invitation token is required.
func (s *Service) Register(ctx context.Context, username, password, invite string) (*gamification.Player, string, error) {
	username = strings.TrimSpace(username)
	if n := len([]rune(username)); n < 2 || n > 20 {
		return nil, "", ErrBadUsername
	}
	if len(password) < 6 {
		return nil, "", ErrWeakPassword
	}

	if _, err := s.players.GetByUsername(ctx, username); err == nil {
		return nil, "", ErrUsernameTaken
	} else if !errors.Is(err, ports.ErrNotFound) {
		return nil, "", err
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, "", err
	}
	id := uuid.New().String()

	// In invite mode, atomically validate AND consume the invitation before
	// creating anything, so a link is strictly single-use and never fails open.
	if s.mode == Invite {
		now := s.now().UTC()
		minCreated := now.Add(-inviteTTL).Format(time.RFC3339)
		ok, err := s.invites.ConsumeInvite(ctx, invite, id, now.Format(time.RFC3339), minCreated)
		if err != nil {
			return nil, "", err
		}
		if !ok {
			return nil, "", ErrInviteRequired
		}
	}

	player, err := gamification.NewPlayer(id, username)
	if err != nil {
		return nil, "", err
	}
	if err := s.players.Create(ctx, player); err != nil {
		return nil, "", err
	}
	if err := s.accounts.SetCredentials(ctx, player.ID(), hash); err != nil {
		return nil, "", err
	}

	return player, auth.IssueToken(player.ID(), s.secret, sessionTTL, s.now()), nil
}

// Login verifies a username/password and returns the player with a token.
func (s *Service) Login(ctx context.Context, username, password string) (*gamification.Player, string, error) {
	acc, err := s.accounts.Credentials(ctx, strings.TrimSpace(username))
	if err != nil || acc.PasswordHash == "" || !auth.CheckPassword(acc.PasswordHash, password) {
		// Uniform error: never reveal whether the username exists.
		return nil, "", ErrInvalidCredentials
	}
	player, err := s.players.GetByID(ctx, acc.ID)
	if err != nil {
		return nil, "", err
	}
	return player, auth.IssueToken(acc.ID, s.secret, sessionTTL, s.now()), nil
}

// Authenticate validates a session token and returns the player id.
func (s *Service) Authenticate(token string) (string, error) {
	id, err := auth.VerifyToken(token, s.secret, s.now())
	if err != nil {
		return "", ErrInvalidCredentials
	}
	return id, nil
}

// IssueInvite mints and stores an invitation, returning its token (admin only).
func (s *Service) IssueInvite(ctx context.Context, createdBy string) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := s.invites.CreateInvite(ctx, token, createdBy, s.now().UTC().Format(time.RFC3339)); err != nil {
		return "", err
	}
	return token, nil
}

// ListInvites returns the issued invitations (admin only).
func (s *Service) ListInvites(ctx context.Context) ([]ports.Invite, error) {
	return s.invites.ListInvites(ctx, 100)
}

// RevokeInvite disables an invitation (admin only).
func (s *Service) RevokeInvite(ctx context.Context, token string) error {
	return s.invites.RevokeInvite(ctx, token)
}

// randomToken returns an unguessable invitation token.
func randomToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating invite token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

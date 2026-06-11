// Package account provides player registration and login: real accounts with
// passwords, with either open self-service signup or admin-invite-only signup.
package account

import (
	"context"
	"errors"
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
	inviteTTL  = 14 * 24 * time.Hour
	inviteSub  = "invite" // token subject for invitations
)

// Service registers and authenticates players.
type Service struct {
	accounts ports.AccountStore
	players  ports.PlayerRepository
	secret   string
	mode     Mode
	now      func() time.Time
}

// NewService creates the account service. mode selects open or invite signup.
func NewService(accounts ports.AccountStore, players ports.PlayerRepository, secret string, mode Mode) *Service {
	if mode != Invite {
		mode = Open
	}
	return &Service{accounts: accounts, players: players, secret: secret, mode: mode, now: time.Now}
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
	if s.mode == Invite && !s.validInvite(invite) {
		return nil, "", ErrInviteRequired
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

	player, err := gamification.NewPlayer(uuid.New().String(), username)
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
	if err != nil || id == inviteSub {
		return "", ErrInvalidCredentials
	}
	return id, nil
}

// IssueInvite mints an invitation token (admin only).
func (s *Service) IssueInvite() string {
	return auth.IssueToken(inviteSub, s.secret, inviteTTL, s.now())
}

// validInvite reports whether an invitation token is genuine and unexpired.
func (s *Service) validInvite(token string) bool {
	sub, err := auth.VerifyToken(token, s.secret, s.now())
	return err == nil && sub == inviteSub
}

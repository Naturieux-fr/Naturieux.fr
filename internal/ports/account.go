package ports

import "context"

// Account holds the authentication data for a player.
type Account struct {
	ID           string
	PasswordHash string
	Role         string
}

// AccountStore provides access to authentication data and admin seeding.
type AccountStore interface {
	// Credentials returns the authentication data for a username.
	Credentials(ctx context.Context, username string) (Account, error)

	// Role returns a player's role.
	Role(ctx context.Context, id string) (string, error)

	// UpsertAdmin creates or updates an admin account.
	UpsertAdmin(ctx context.Context, id, username, passwordHash, createdAt string) error

	// SetCredentials sets a player's password hash (used at registration).
	SetCredentials(ctx context.Context, id, passwordHash string) error
}

// PlayerSummary is a compact view of a player for the admin back-office.
type PlayerSummary struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	Role       string  `json:"role"`
	Level      int     `json:"level"`
	TotalGames int     `json:"total_games"`
	Accuracy   float64 `json:"accuracy"`
	CreatedAt  string  `json:"created_at"`
}

// PlayerAdminStore exposes player administration queries.
type PlayerAdminStore interface {
	CountPlayers(ctx context.Context) (int, error)
	CountAdmins(ctx context.Context) (int, error)
	TotalGames(ctx context.Context) (int, error)
	ListPlayers(ctx context.Context, limit int) ([]PlayerSummary, error)
	Role(ctx context.Context, id string) (string, error)
	DeletePlayer(ctx context.Context, id string) error
	SetRole(ctx context.Context, id, role string) error
}

// Invite is a stored registration invitation.
type Invite struct {
	Token     string `json:"token"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
	UsedBy    string `json:"used_by,omitempty"`
	UsedAt    string `json:"used_at,omitempty"`
	Revoked   bool   `json:"revoked"`
}

// InviteStore persists registration invitations for tracking and revocation.
type InviteStore interface {
	CreateInvite(ctx context.Context, token, createdBy, createdAt string) error
	GetInvite(ctx context.Context, token string) (Invite, error)
	// ConsumeInvite atomically validates and consumes an invite: it marks the
	// token used only if it is still pending (not used, not revoked) and was
	// created at or after minCreatedAt. It reports whether a row was consumed.
	ConsumeInvite(ctx context.Context, token, usedBy, usedAt, minCreatedAt string) (bool, error)
	RevokeInvite(ctx context.Context, token string) error
	ListInvites(ctx context.Context, limit int) ([]Invite, error)
}

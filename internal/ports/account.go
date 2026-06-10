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
}

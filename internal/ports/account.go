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
	TotalGames(ctx context.Context) (int, error)
	ListPlayers(ctx context.Context, limit int) ([]PlayerSummary, error)
	DeletePlayer(ctx context.Context, id string) error
	SetRole(ctx context.Context, id, role string) error
}

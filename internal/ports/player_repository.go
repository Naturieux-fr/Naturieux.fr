package ports

import (
	"context"

	"github.com/fieve/naturieux/internal/domain/gamification"
)

// PlayerRepository defines the interface for player data persistence.
type PlayerRepository interface {
	// Create creates a new player.
	Create(ctx context.Context, player *gamification.Player) error

	// GetByID retrieves a player by ID.
	GetByID(ctx context.Context, id string) (*gamification.Player, error)

	// GetByUsername retrieves a player by username.
	GetByUsername(ctx context.Context, username string) (*gamification.Player, error)

	// Update updates a player's data.
	Update(ctx context.Context, player *gamification.Player) error

	// GetLeaderboard retrieves top players by XP.
	GetLeaderboard(ctx context.Context, limit int) ([]*gamification.Player, error)
}

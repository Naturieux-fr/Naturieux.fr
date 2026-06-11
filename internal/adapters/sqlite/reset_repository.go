package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

// ResetRepository stores admin-issued, single-use password-reset tokens.
type ResetRepository struct {
	db *sql.DB
}

// NewResetRepository creates a password-reset repository.
func NewResetRepository(db *sql.DB) *ResetRepository {
	return &ResetRepository{db: db}
}

// CreateReset stores a new pending reset token for a player.
func (r *ResetRepository) CreateReset(ctx context.Context, token, playerID, createdAt string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO password_resets (token, player_id, created_at) VALUES (?, ?, ?)`,
		token, playerID, createdAt)
	if err != nil {
		return fmt.Errorf("creating reset: %w", err)
	}
	return nil
}

// ConsumeReset atomically marks a reset token used if it is still pending and
// not older than minCreatedAt, returning the player id it belongs to. ok is
// false when the token is unknown, already used or expired.
func (r *ResetRepository) ConsumeReset(ctx context.Context, token, minCreatedAt string) (string, bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE password_resets SET used = 1 WHERE token = ? AND used = 0 AND created_at >= ?`,
		token, minCreatedAt)
	if err != nil {
		return "", false, fmt.Errorf("consuming reset: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return "", false, nil
	}
	var playerID string
	if err := r.db.QueryRowContext(ctx, `SELECT player_id FROM password_resets WHERE token = ?`, token).Scan(&playerID); err != nil {
		return "", false, fmt.Errorf("loading reset player: %w", err)
	}
	return playerID, true, nil
}

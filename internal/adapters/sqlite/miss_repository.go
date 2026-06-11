package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// MissRepository tracks the species a player gets wrong, to power the revision
// mode (light spaced repetition).
type MissRepository struct {
	db *sql.DB
}

// NewMissRepository creates a miss repository.
func NewMissRepository(db *sql.DB) *MissRepository {
	return &MissRepository{db: db}
}

// RecordMiss increments the miss count for a species the player got wrong.
func (r *MissRepository) RecordMiss(ctx context.Context, playerID string, cdNom int) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO player_misses (player_id, cd_nom, misses, last_at)
		VALUES (?, ?, 1, ?)
		ON CONFLICT(player_id, cd_nom) DO UPDATE SET
			misses = player_misses.misses + 1, last_at = excluded.last_at`,
		playerID, cdNom, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("recording miss: %w", err)
	}
	return nil
}

// DecayMiss decrements a species' miss count when the player gets it right,
// removing it from the revision pool once mastered.
func (r *MissRepository) DecayMiss(ctx context.Context, playerID string, cdNom int) error {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE player_misses SET misses = misses - 1 WHERE player_id = ? AND cd_nom = ?`,
		playerID, cdNom); err != nil {
		return fmt.Errorf("decaying miss: %w", err)
	}
	_, _ = r.db.ExecContext(ctx, `DELETE FROM player_misses WHERE player_id = ? AND cd_nom = ? AND misses <= 0`, playerID, cdNom)
	return nil
}

// TopMisses returns the species ids the player most needs to revise, most
// missed first.
func (r *MissRepository) TopMisses(ctx context.Context, playerID string, limit int) ([]int, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT cd_nom FROM player_misses WHERE player_id = ? AND misses > 0
		ORDER BY misses DESC, last_at DESC LIMIT ?`, playerID, limit)
	if err != nil {
		return nil, fmt.Errorf("loading misses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]int, 0, limit)
	for rows.Next() {
		var cd int
		if err := rows.Scan(&cd); err != nil {
			return nil, fmt.Errorf("scanning miss: %w", err)
		}
		out = append(out, cd)
	}
	return out, rows.Err()
}

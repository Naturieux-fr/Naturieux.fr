package sqlite

import (
	"context"
	"fmt"

	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// CountPlayers returns the number of registered players.
func (r *PlayerRepository) CountPlayers(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM players`).Scan(&n); err != nil {
		return 0, fmt.Errorf("counting players: %w", err)
	}
	return n, nil
}

// CountAdmins returns the number of admin accounts.
func (r *PlayerRepository) CountAdmins(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM players WHERE role = 'admin'`).Scan(&n); err != nil {
		return 0, fmt.Errorf("counting admins: %w", err)
	}
	return n, nil
}

// TotalGames returns the total number of games played across all players.
func (r *PlayerRepository) TotalGames(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(total_games), 0) FROM players`).Scan(&n); err != nil {
		return 0, fmt.Errorf("summing games: %w", err)
	}
	return n, nil
}

// ListPlayers returns compact player rows, newest games first.
func (r *PlayerRepository) ListPlayers(ctx context.Context, limit int) ([]ports.PlayerSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, username, role, level, total_games, total_correct, total_questions, created_at
		FROM players ORDER BY total_games DESC, total_xp DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("listing players: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]ports.PlayerSummary, 0, limit)
	for rows.Next() {
		var s ports.PlayerSummary
		var correct, questions int
		if err := rows.Scan(&s.ID, &s.Username, &s.Role, &s.Level, &s.TotalGames, &correct, &questions, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning player summary: %w", err)
		}
		if questions > 0 {
			s.Accuracy = float64(correct) / float64(questions) * 100
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DeletePlayer removes a player account.
func (r *PlayerRepository) DeletePlayer(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM players WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting player: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// SetRole changes a player's role ("player" or "admin").
func (r *PlayerRepository) SetRole(ctx context.Context, id, role string) error {
	if role != "player" && role != "admin" {
		return fmt.Errorf("invalid role %q", role)
	}
	res, err := r.db.ExecContext(ctx, `UPDATE players SET role = ? WHERE id = ?`, role, id)
	if err != nil {
		return fmt.Errorf("setting role: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// Ensure interface compliance.
var _ ports.PlayerAdminStore = (*PlayerRepository)(nil)

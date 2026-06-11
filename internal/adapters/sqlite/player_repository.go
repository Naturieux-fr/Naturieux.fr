package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// PlayerRepository is a SQLite implementation of ports.PlayerRepository.
type PlayerRepository struct {
	db *sql.DB
}

// NewPlayerRepository creates a new SQLite player repository.
func NewPlayerRepository(db *sql.DB) *PlayerRepository {
	return &PlayerRepository{db: db}
}

const playerColumns = `id, username, total_xp, level, total_games, total_correct,
	total_questions, best_streak, daily_streak, achievements, last_played_at, created_at,
	category_correct`

// Create creates a new player.
func (r *PlayerRepository) Create(ctx context.Context, player *gamification.Player) error {
	snap := player.Snapshot()

	achievements, err := json.Marshal(snap.Achievements)
	if err != nil {
		return fmt.Errorf("encoding achievements: %w", err)
	}

	categoryCorrect, err := json.Marshal(snap.CategoryCorrect)
	if err != nil {
		return fmt.Errorf("encoding category counts: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO players (`+playerColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.ID, snap.Username, snap.TotalXP, snap.Level, snap.TotalGames,
		snap.TotalCorrect, snap.TotalQuestions, snap.BestStreak, snap.DailyStreak,
		string(achievements), formatNullableTime(snap.LastPlayedAt),
		snap.CreatedAt.Format(time.RFC3339Nano), string(categoryCorrect),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ports.ErrAlreadyExists
		}
		return fmt.Errorf("inserting player: %w", err)
	}
	return nil
}

// GetByID retrieves a player by ID.
func (r *PlayerRepository) GetByID(ctx context.Context, id string) (*gamification.Player, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+playerColumns+` FROM players WHERE id = ?`, id)
	return scanPlayer(row)
}

// GetByUsername retrieves a player by username.
func (r *PlayerRepository) GetByUsername(ctx context.Context, username string) (*gamification.Player, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+playerColumns+` FROM players WHERE username = ?`, username)
	return scanPlayer(row)
}

// Update updates a player's data.
func (r *PlayerRepository) Update(ctx context.Context, player *gamification.Player) error {
	snap := player.Snapshot()

	achievements, err := json.Marshal(snap.Achievements)
	if err != nil {
		return fmt.Errorf("encoding achievements: %w", err)
	}

	categoryCorrect, err := json.Marshal(snap.CategoryCorrect)
	if err != nil {
		return fmt.Errorf("encoding category counts: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE players SET username = ?, total_xp = ?, level = ?, total_games = ?,
			total_correct = ?, total_questions = ?, best_streak = ?, daily_streak = ?,
			achievements = ?, last_played_at = ?, category_correct = ?
		WHERE id = ?`,
		snap.Username, snap.TotalXP, snap.Level, snap.TotalGames,
		snap.TotalCorrect, snap.TotalQuestions, snap.BestStreak, snap.DailyStreak,
		string(achievements), formatNullableTime(snap.LastPlayedAt), string(categoryCorrect), snap.ID,
	)
	if err != nil {
		return fmt.Errorf("updating player: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking update result: %w", err)
	}
	if affected == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// GetLeaderboard retrieves top players by XP.
func (r *PlayerRepository) GetLeaderboard(ctx context.Context, limit int) ([]*gamification.Player, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+playerColumns+` FROM players ORDER BY total_xp DESC, username ASC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("querying leaderboard: %w", err)
	}
	defer func() { _ = rows.Close() }()

	players := make([]*gamification.Player, 0, limit)
	for rows.Next() {
		player, err := scanPlayer(rows)
		if err != nil {
			return nil, err
		}
		players = append(players, player)
	}
	return players, rows.Err()
}

// rowScanner abstracts *sql.Row and *sql.Rows for scanning.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanPlayer reads a player row and reconstructs the domain entity.
func scanPlayer(row rowScanner) (*gamification.Player, error) {
	var snap gamification.PlayerSnapshot
	var achievements string
	var categoryCorrect sql.NullString
	var lastPlayedAt sql.NullString
	var createdAt string

	err := row.Scan(
		&snap.ID, &snap.Username, &snap.TotalXP, &snap.Level, &snap.TotalGames,
		&snap.TotalCorrect, &snap.TotalQuestions, &snap.BestStreak, &snap.DailyStreak,
		&achievements, &lastPlayedAt, &createdAt, &categoryCorrect,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning player: %w", err)
	}

	if err := json.Unmarshal([]byte(achievements), &snap.Achievements); err != nil {
		return nil, fmt.Errorf("decoding achievements: %w", err)
	}
	if categoryCorrect.Valid && categoryCorrect.String != "" {
		if err := json.Unmarshal([]byte(categoryCorrect.String), &snap.CategoryCorrect); err != nil {
			return nil, fmt.Errorf("decoding category counts: %w", err)
		}
	}
	if snap.LastPlayedAt, err = parseNullableTime(lastPlayedAt); err != nil {
		return nil, fmt.Errorf("parsing last_played_at: %w", err)
	}
	if snap.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return nil, fmt.Errorf("parsing created_at: %w", err)
	}

	return gamification.RestorePlayer(snap)
}

// formatNullableTime converts an optional time to a nullable string.
func formatNullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}

// parseNullableTime converts a nullable string back to an optional time.
func parseNullableTime(s sql.NullString) (*time.Time, error) {
	if !s.Valid {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Ensure interface compliance.
var _ ports.PlayerRepository = (*PlayerRepository)(nil)

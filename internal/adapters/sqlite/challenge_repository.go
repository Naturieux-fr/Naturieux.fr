package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

// ChallengeRepository persists per-challenge scores and leaderboards.
type ChallengeRepository struct {
	db *sql.DB
}

// NewChallengeRepository creates a challenge score repository.
func NewChallengeRepository(db *sql.DB) *ChallengeRepository {
	return &ChallengeRepository{db: db}
}

// ChallengeEntry is one row of a challenge leaderboard.
type ChallengeEntry struct {
	Rank     int    `json:"rank"`
	Username string `json:"username"`
	Score    int    `json:"score"`
	Correct  int    `json:"correct"`
	Total    int    `json:"total"`
}

// SaveScore records a player's challenge result, keeping only their best score
// for that challenge.
func (r *ChallengeRepository) SaveScore(ctx context.Context, period, key, playerID, username string, score, correct, total int, playedAt string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO challenge_scores (period, period_key, player_id, username, score, correct, total, played_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(period, period_key, player_id) DO UPDATE SET
			correct   = CASE WHEN excluded.score > challenge_scores.score THEN excluded.correct ELSE challenge_scores.correct END,
			total     = excluded.total,
			username  = excluded.username,
			played_at = CASE WHEN excluded.score > challenge_scores.score THEN excluded.played_at ELSE challenge_scores.played_at END,
			score     = MAX(challenge_scores.score, excluded.score)`,
		period, key, playerID, username, score, correct, total, playedAt)
	if err != nil {
		return fmt.Errorf("saving challenge score: %w", err)
	}
	return nil
}

// Leaderboard returns the top scores for a challenge.
func (r *ChallengeRepository) Leaderboard(ctx context.Context, period, key string, limit int) ([]ChallengeEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT username, score, correct, total FROM challenge_scores
		WHERE period = ? AND period_key = ?
		ORDER BY score DESC, played_at ASC LIMIT ?`, period, key, limit)
	if err != nil {
		return nil, fmt.Errorf("challenge leaderboard: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]ChallengeEntry, 0, limit)
	rank := 0
	for rows.Next() {
		rank++
		e := ChallengeEntry{Rank: rank}
		if err := rows.Scan(&e.Username, &e.Score, &e.Correct, &e.Total); err != nil {
			return nil, fmt.Errorf("scanning challenge entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// BestFor returns a player's best score for a challenge and whether they played.
func (r *ChallengeRepository) BestFor(ctx context.Context, period, key, playerID string) (int, bool, error) {
	var score int
	err := r.db.QueryRowContext(ctx,
		`SELECT score FROM challenge_scores WHERE period = ? AND period_key = ? AND player_id = ?`,
		period, key, playerID).Scan(&score)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("challenge best: %w", err)
	}
	return score, true, nil
}

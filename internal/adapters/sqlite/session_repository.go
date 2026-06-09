package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// SessionRepository is a SQLite implementation of ports.QuizSessionRepository.
// The full session state is stored as JSON for reconstruction, alongside
// denormalized columns used for statistics queries.
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new SQLite session repository.
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Save persists a quiz session (insert or update).
func (r *SessionRepository) Save(ctx context.Context, session *quiz.Session) error {
	snap := session.Snapshot()

	data, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("encoding session: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO quiz_sessions (id, user_id, status, difficulty, taxon_filter,
			total_score, max_streak, questions_count, correct_count,
			started_at, completed_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			total_score = excluded.total_score,
			max_streak = excluded.max_streak,
			correct_count = excluded.correct_count,
			completed_at = excluded.completed_at,
			data = excluded.data`,
		snap.ID, snap.UserID, string(snap.Status), string(snap.Difficulty),
		snap.TaxonFilter, snap.TotalScore, snap.MaxStreak, len(snap.Questions),
		session.CorrectCount(), snap.StartedAt.Format(time.RFC3339Nano),
		formatNullableTime(snap.CompletedAt), string(data),
	)
	if err != nil {
		return fmt.Errorf("saving session: %w", err)
	}
	return nil
}

// GetByID retrieves a session by ID.
func (r *SessionRepository) GetByID(ctx context.Context, id string) (*quiz.Session, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM quiz_sessions WHERE id = ?`, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying session: %w", err)
	}
	return restoreFromJSON(data)
}

// GetByUserID retrieves the most recent sessions for a user.
func (r *SessionRepository) GetByUserID(ctx context.Context, userID string, limit int) ([]*quiz.Session, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT data FROM quiz_sessions
		WHERE user_id = ?
		ORDER BY started_at DESC
		LIMIT ?`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	sessions := make([]*quiz.Session, 0, limit)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("scanning session: %w", err)
		}
		session, err := restoreFromJSON(data)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

// GetStats retrieves aggregated stats for a user's completed sessions.
func (r *SessionRepository) GetStats(ctx context.Context, userID string) (*ports.UserQuizStats, error) {
	stats := &ports.UserQuizStats{}

	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*),
			COALESCE(SUM(questions_count), 0),
			COALESCE(SUM(correct_count), 0),
			COALESCE(SUM(total_score), 0),
			COALESCE(MAX(max_streak), 0)
		FROM quiz_sessions
		WHERE user_id = ? AND status = ?`,
		userID, string(quiz.SessionCompleted),
	).Scan(&stats.TotalSessions, &stats.TotalQuestions, &stats.TotalCorrect,
		&stats.TotalScore, &stats.BestStreak)
	if err != nil {
		return nil, fmt.Errorf("querying stats: %w", err)
	}

	if stats.TotalQuestions > 0 {
		stats.AverageAccuracy = float64(stats.TotalCorrect) / float64(stats.TotalQuestions) * 100
	}

	stats.FavoriteTaxon, err = r.favoriteTaxon(ctx, userID)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// favoriteTaxon returns the taxon filter the user plays most often.
func (r *SessionRepository) favoriteTaxon(ctx context.Context, userID string) (string, error) {
	var taxon string
	err := r.db.QueryRowContext(ctx, `
		SELECT taxon_filter FROM quiz_sessions
		WHERE user_id = ? AND taxon_filter != ''
		GROUP BY taxon_filter
		ORDER BY COUNT(*) DESC
		LIMIT 1`, userID).Scan(&taxon)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("querying favorite taxon: %w", err)
	}
	return taxon, nil
}

// restoreFromJSON rebuilds a domain session from its stored snapshot.
func restoreFromJSON(data string) (*quiz.Session, error) {
	var snap quiz.SessionSnapshot
	if err := json.Unmarshal([]byte(data), &snap); err != nil {
		return nil, fmt.Errorf("decoding session: %w", err)
	}
	return quiz.RestoreSession(snap)
}

// Ensure interface compliance.
var _ ports.QuizSessionRepository = (*SessionRepository)(nil)
